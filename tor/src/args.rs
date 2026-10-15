//! Command-line options of the `omnitun-tor` binary.

use std::net::SocketAddr;
use std::path::PathBuf;

/// Parsed command line.
#[derive(Debug, Default, PartialEq)]
pub struct Args {
    /// Data directory for Arti's state and cache.
    pub data_dir: PathBuf,
    /// SOCKS5 listen port on 127.0.0.1 (0 = any free port).
    pub socks_port: u16,
    /// Bridge lines, e.g. "obfs4 192.0.2.1:443 FINGERPRINT cert=... iat-mode=0".
    pub bridges: Vec<String>,
    /// Unmanaged pluggable transports: (protocol, local SOCKS address).
    pub transports: Vec<(String, SocketAddr)>,
}

/// Parses `--data DIR --socks-port N [--bridge LINE]... [--transport NAME=ADDR]...`.
pub fn parse(mut it: impl Iterator<Item = String>) -> Result<Args, String> {
    let mut a = Args::default();
    let mut have_data = false;
    while let Some(flag) = it.next() {
        let mut value = || it.next().ok_or_else(|| format!("{flag} needs a value"));
        match flag.as_str() {
            "--data" => {
                a.data_dir = PathBuf::from(value()?);
                have_data = true;
            }
            "--socks-port" => {
                a.socks_port = value()?.parse().map_err(|e| format!("--socks-port: {e}"))?;
            }
            "--bridge" => {
                let line = value()?;
                // Accept torrc style "Bridge obfs4 ..." too.
                let line = line.strip_prefix("Bridge ").unwrap_or(&line).trim().to_string();
                if !line.is_empty() {
                    a.bridges.push(line);
                }
            }
            "--transport" => {
                let v = value()?;
                let (name, addr) = v.split_once('=').ok_or("--transport needs NAME=HOST:PORT")?;
                let addr = addr.parse().map_err(|e| format!("--transport {name}: {e}"))?;
                a.transports.push((name.to_string(), addr));
            }
            other => return Err(format!("unknown option {other}")),
        }
    }
    if !have_data {
        return Err("--data is required".into());
    }
    Ok(a)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn args(v: &[&str]) -> Result<Args, String> {
        parse(v.iter().map(|s| s.to_string()))
    }

    #[test]
    fn parses_everything() {
        let a = args(&[
            "--data", "d", "--socks-port", "9150", "--bridge", "Bridge obfs4 192.0.2.1:443 AAAA cert=x iat-mode=0",
            "--transport", "obfs4=127.0.0.1:4000",
        ])
        .unwrap();
        assert_eq!(a.socks_port, 9150);
        assert_eq!(a.bridges, vec!["obfs4 192.0.2.1:443 AAAA cert=x iat-mode=0"]);
        assert_eq!(a.transports, vec![("obfs4".into(), "127.0.0.1:4000".parse().unwrap())]);
    }

    #[test]
    fn rejects_bad_input() {
        assert!(args(&[]).is_err());
        assert!(args(&["--data"]).is_err());
        assert!(args(&["--data", "d", "--transport", "obfs4"]).is_err());
        assert!(args(&["--data", "d", "--nope"]).is_err());
    }
}
