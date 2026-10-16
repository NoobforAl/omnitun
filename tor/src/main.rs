//! `omnitun-tor`: Arti behind a local SOCKS5 port, started by the Go core.
//!
//! Usage: omnitun-tor --data DIR [--socks-port N] [--bridge LINE]... [--transport NAME=HOST:PORT]...
//!
//! stdout carries one JSON object per line for the core:
//!   {"event":"listening","socks":"127.0.0.1:PORT"}
//!   {"event":"bootstrap","percent":42,"status":"..."}
//!   {"event":"ready"}
//!   {"event":"error","message":"..."}
//! The process exits when stdin closes, so it never outlives the core.

use std::sync::Arc;

use anyhow::Context;
use arti_client::config::pt::TransportConfigBuilder;
use arti_client::config::{BoolOrAuto, BridgeConfigBuilder, TorClientConfigBuilder};
use arti_client::{HasKind, TorClient, TorClientConfig};
use futures::StreamExt;
use omnitun_tor::{args, socks};
use serde_json::json;
use tokio::io::{AsyncReadExt, AsyncWriteExt};
use tokio::net::{TcpListener, TcpStream};
use tor_rtcompat::PreferredRuntime;

fn emit(v: serde_json::Value) {
    // One line per event; stdout is line-buffered by the core's reader.
    println!("{v}");
}

fn config(a: &args::Args) -> anyhow::Result<TorClientConfig> {
    let mut b = TorClientConfigBuilder::from_directories(
        a.data_dir.join("state"),
        a.data_dir.join("cache"),
    );
    for line in &a.bridges {
        let bridge: BridgeConfigBuilder = line
            .parse()
            .with_context(|| format!("bad bridge line {line:?}"))?;
        b.bridges().bridges().push(bridge);
    }
    if !a.bridges.is_empty() {
        b.bridges().enabled(BoolOrAuto::Explicit(true));
    }
    for (name, addr) in &a.transports {
        let mut t = TransportConfigBuilder::default();
        t.protocols(vec![
            name.parse()
                .with_context(|| format!("bad transport name {name:?}"))?,
        ])
        .proxy_addr(*addr);
        b.bridges().transports().push(t);
    }
    Ok(b.build()?)
}

async fn serve(client: Arc<TorClient<PreferredRuntime>>, mut conn: TcpStream) {
    let target = match socks::handshake(&mut conn).await {
        Ok(t) => t,
        Err(_) => return,
    };
    match client.connect((target.host.as_str(), target.port)).await {
        Ok(mut stream) => {
            if socks::send_reply(&mut conn, socks::reply::SUCCEEDED)
                .await
                .is_err()
            {
                return;
            }
            let _ = tokio::io::copy_bidirectional(&mut conn, &mut stream).await;
        }
        Err(e) => {
            // Never log the destination: the app log can be shown or exported.
            eprintln!("connect failed: {:?}", e.kind());
            let code = match e.kind() {
                arti_client::ErrorKind::RemoteHostNotFound
                | arti_client::ErrorKind::RemoteNetworkFailed => socks::reply::HOST_UNREACHABLE,
                _ => socks::reply::GENERAL_FAILURE,
            };
            let _ = socks::send_reply(&mut conn, code).await;
        }
    }
}

async fn run(a: args::Args) -> anyhow::Result<()> {
    let listener = TcpListener::bind(("127.0.0.1", a.socks_port))
        .await
        .context("listen")?;
    emit(json!({"event": "listening", "socks": listener.local_addr()?.to_string()}));

    let client = TorClient::builder()
        .config(config(&a)?)
        .create_unbootstrapped()?;
    let mut events = client.bootstrap_events();
    tokio::spawn(async move {
        let mut last = -1;
        while let Some(status) = events.next().await {
            let percent = (status.as_frac() * 100.0) as i32;
            if percent != last {
                last = percent;
                let mut v =
                    json!({"event": "bootstrap", "percent": percent, "status": status.to_string()});
                if let Some(b) = status.blocked() {
                    v["blocked"] = json!(b.to_string());
                }
                emit(v);
            }
        }
    });
    client.bootstrap().await.context("bootstrap")?;
    emit(json!({"event": "ready"}));

    loop {
        let (conn, _) = listener.accept().await?;
        let client = client.clone();
        tokio::spawn(serve(client, conn));
    }
}

#[tokio::main]
async fn main() {
    let _ = rustls::crypto::ring::default_provider().install_default();
    // Arti's logs go to stderr (RUST_LOG overrides the default "warn").
    tracing_subscriber::fmt()
        .with_writer(std::io::stderr)
        .with_ansi(false)
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env().unwrap_or_else(|_| {
                "warn,tor_circmgr=info,tor_guardmgr=info,tor_chanmgr=info".into()
            }),
        )
        .init();
    let a = match args::parse(std::env::args().skip(1)) {
        Ok(a) => a,
        Err(e) => {
            emit(json!({"event": "error", "message": e}));
            std::process::exit(2);
        }
    };
    // Exit when the core goes away (stdin closes).
    tokio::spawn(async {
        let mut stdin = tokio::io::stdin();
        let mut buf = [0u8; 64];
        while let Ok(n) = stdin.read(&mut buf).await {
            if n == 0 {
                break;
            }
        }
        std::process::exit(0);
    });
    if let Err(e) = run(a).await {
        emit(json!({"event": "error", "message": format!("{e:#}")}));
        let _ = tokio::io::stdout().flush().await;
        std::process::exit(1);
    }
}
