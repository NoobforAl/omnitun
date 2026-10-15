//! C ABI for the Tor module. The Go core calls these functions; Tor traffic is
//! exposed as a local SOCKS5 port that the core routes the tunnel into.
//!
//! Arti itself is wired in Phase 6 behind the `arti` feature.

use std::ffi::{CStr, c_char};
use std::sync::Mutex;

/// Error codes returned by the ABI (negative values).
pub mod errors {
    pub const INVALID_ARGUMENT: i32 = -1;
    pub const ALREADY_RUNNING: i32 = -2;
    pub const NOT_IMPLEMENTED: i32 = -100;
}

/// Running client state. Holds the SOCKS port while started.
static STATE: Mutex<Option<u16>> = Mutex::new(None);

/// Returns the crate version as a static NUL-terminated string.
#[unsafe(no_mangle)]
pub extern "C" fn vt_tor_version() -> *const c_char {
    concat!(env!("CARGO_PKG_VERSION"), "\0").as_ptr().cast()
}

/// Starts the Tor client.
///
/// `config_json` is a UTF-8 JSON object: `{"socks_port":9050,"data_dir":"...","bridges":["obfs4 ..."]}`.
/// Returns the SOCKS port (> 0) on success or a negative error code.
///
/// # Safety
/// `config_json` must be null or point to a valid NUL-terminated string.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn vt_tor_start(config_json: *const c_char) -> i32 {
    if config_json.is_null() {
        return errors::INVALID_ARGUMENT;
    }
    // SAFETY: caller guarantees a valid NUL-terminated string.
    let Ok(_config) = unsafe { CStr::from_ptr(config_json) }.to_str() else {
        return errors::INVALID_ARGUMENT;
    };
    let state = STATE.lock().unwrap_or_else(|e| e.into_inner());
    if state.is_some() {
        return errors::ALREADY_RUNNING;
    }
    errors::NOT_IMPLEMENTED
}

/// Stops the Tor client. Safe to call when it is not running.
#[unsafe(no_mangle)]
pub extern "C" fn vt_tor_stop() {
    STATE.lock().unwrap_or_else(|e| e.into_inner()).take();
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn version_is_c_string() {
        let v = unsafe { CStr::from_ptr(vt_tor_version()) };
        assert_eq!(v.to_str().unwrap(), env!("CARGO_PKG_VERSION"));
    }

    #[test]
    fn start_rejects_null() {
        assert_eq!(unsafe { vt_tor_start(std::ptr::null()) }, errors::INVALID_ARGUMENT);
    }

    #[test]
    fn start_is_not_implemented_yet() {
        let cfg = c"{\"socks_port\":9050}";
        assert_eq!(unsafe { vt_tor_start(cfg.as_ptr()) }, errors::NOT_IMPLEMENTED);
        vt_tor_stop();
    }
}
