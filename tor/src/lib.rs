//! Omnitun's Tor client: Arti behind a minimal local SOCKS5 server.
//!
//! The `omnitun-tor` binary (src/main.rs) is started by the Go core, which
//! routes Tor profiles into its SOCKS port. Bridges use pluggable transports
//! that the core runs in-process (obfs4, Snowflake, WebTunnel) and passes in
//! as "unmanaged" transports (`--transport obfs4=127.0.0.1:PORT`).

pub mod args;
pub mod socks;
