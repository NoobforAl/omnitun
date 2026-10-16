// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for English (`en`).
class AppLocalizationsEn extends AppLocalizations {
  AppLocalizationsEn([String locale = 'en']) : super(locale);

  @override
  String get appName => 'Omnitun';

  @override
  String get actionSave => 'Save';

  @override
  String get actionCancel => 'Cancel';

  @override
  String get actionDelete => 'Delete';

  @override
  String get actionEdit => 'Edit';

  @override
  String get actionReset => 'Reset';

  @override
  String get actionSkip => 'Skip';

  @override
  String get actionNext => 'Next';

  @override
  String get actionImport => 'Import';

  @override
  String get actionDownload => 'Download';

  @override
  String get actionCopyAll => 'Copy all';

  @override
  String get actionClear => 'Clear';

  @override
  String get tooltipMore => 'More';

  @override
  String get tooltipShow => 'Show';

  @override
  String get tooltipHide => 'Hide';

  @override
  String get searchHint => 'Search';

  @override
  String get tooltipLogs => 'Logs';

  @override
  String get tooltipSettings => 'Settings';

  @override
  String get statusConnected => 'Connected';

  @override
  String get statusConnecting => 'Connecting…';

  @override
  String get statusDisconnecting => 'Disconnecting…';

  @override
  String get statusError => 'Connection failed';

  @override
  String get statusNoInternet => 'No internet connection';

  @override
  String get statusDisconnected => 'Not connected';

  @override
  String get stagePermission => 'Waiting for VPN permission';

  @override
  String get stageNetworkCheck => 'Checking internet connection';

  @override
  String get stagePrepare => 'Preparing configuration';

  @override
  String get stageEngineStart => 'Starting engine';

  @override
  String get stageTunnelUp => 'Opening tunnel';

  @override
  String get stageHandshake => 'Contacting server';

  @override
  String get stageConnected => 'Connected';

  @override
  String get buttonConnect => 'Connect';

  @override
  String get buttonDisconnect => 'Disconnect';

  @override
  String get buttonCancel => 'Cancel';

  @override
  String get locationLabel => 'Location';

  @override
  String get noServers => 'No servers';

  @override
  String get noServersHint => 'Add one to get started';

  @override
  String get automatic => 'Automatic';

  @override
  String get automaticHint => 'Fastest server, switches if one stops working';

  @override
  String get anyServerInCountry => 'Any server in this country';

  @override
  String get torNetwork => 'Tor network';

  @override
  String get emptyTitle => 'No profiles yet';

  @override
  String get emptyHint =>
      'Import a config file, link or subscription to get started.';

  @override
  String get importConfig => 'Import config';

  @override
  String failoverToast(String to, String from) {
    return 'Switched to $to — $from stopped responding';
  }

  @override
  String get unknownLocation => 'Unknown location';

  @override
  String errorFailed(String stage, String reason) {
    return '$stage failed: $reason';
  }

  @override
  String get errTimeout => 'the server did not answer in time';

  @override
  String get errRefused => 'the server refused the connection';

  @override
  String get errReset => 'the connection was reset by the network';

  @override
  String get errNoRoute => 'the server cannot be reached from this network';

  @override
  String get errNetworkUnreachable => 'the network is unreachable';

  @override
  String get errDns => 'the server name could not be resolved';

  @override
  String get errTls => 'the server\'s security certificate was rejected';

  @override
  String get errAuth => 'the username or password was rejected';

  @override
  String get errClosed => 'the server closed the connection';

  @override
  String get errTorTimeout =>
      'Tor did not finish connecting in time. This network may block Tor; use a bridge (Snowflake or obfs4)';

  @override
  String get errTorBridgeTimeout =>
      'Tor did not finish connecting in time. The bridge may be blocked or overloaded; try another bridge type (Snowflake or obfs4)';

  @override
  String get errTorBadBridge =>
      'a Tor bridge line is not valid (it needs a transport, address and fingerprint)';

  @override
  String get errTorMissing => 'this build does not include the Tor client';

  @override
  String get errPermission => 'VPN permission was not granted';

  @override
  String get errCoreLost => 'Lost connection to the core service';

  @override
  String get selectLocation => 'Select location';

  @override
  String get testSpeedTooltip => 'Test speed of all servers';

  @override
  String get addServersTooltip => 'Add servers';

  @override
  String get menuImport => 'Import a config, link or QR';

  @override
  String get menuNewServer => 'Add a server by hand';

  @override
  String get menuSubscription => 'Add a subscription';

  @override
  String get menuManage => 'Manage servers and subscriptions';

  @override
  String serversCount(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count servers',
      one: '1 server',
    );
    return '$_temp0';
  }

  @override
  String get otherGroup => 'Other';

  @override
  String get torGroup => 'Tor';

  @override
  String get noServersYet => 'No servers yet';

  @override
  String get importHintShort => 'Import a config file, link or subscription.';

  @override
  String get showServers => 'Show servers';

  @override
  String get hideServers => 'Hide servers';

  @override
  String get menuEditServer => 'Edit';

  @override
  String get menuBridgeSettings => 'Bridge settings';

  @override
  String get menuShare => 'Share link or QR';

  @override
  String get menuSetCountry => 'Set country';

  @override
  String get menuCredentials => 'Username and password';

  @override
  String needsSignIn(String address) {
    return '$address · needs sign-in';
  }

  @override
  String get countryOfServer => 'Country of this server';

  @override
  String get guessFromName => 'Work it out from the name';

  @override
  String get latencyNotTested => 'Not tested';

  @override
  String get latencyUnreachable => 'Unreachable';

  @override
  String get latencyTimeout => 'timeout';

  @override
  String latencyMs(int ms) {
    return '$ms ms';
  }

  @override
  String get unknownCountry => 'Unknown country';

  @override
  String get addServerTitle => 'Add a server';

  @override
  String get addServerIntro =>
      'Choose the protocol of your server. A share link or QR code is quicker: use Import.';

  @override
  String newServerTitle(String label) {
    return 'New $label server';
  }

  @override
  String get editServerTitle => 'Edit server';

  @override
  String get torBridgesTitle => 'Tor bridges';

  @override
  String get subscriptionEditNote =>
      'This server comes from a subscription: your changes are replaced when it refreshes.';

  @override
  String get moreOptions => 'More options';

  @override
  String get countryRow => 'Country';

  @override
  String get countryAuto => 'Worked out from the name and address';

  @override
  String get alreadyInList => 'This server is already in the list';

  @override
  String get choiceDefault => 'Default';

  @override
  String get torBridgesNone => 'None (direct, or the lines below)';

  @override
  String get torBridgesSnowflake => 'Snowflake';

  @override
  String get torBridgesObfs4 => 'obfs4 (built-in bridges)';

  @override
  String get listHelper => 'Separate values with commas';

  @override
  String get fillPaste => 'Paste';

  @override
  String get fillFile => 'File';

  @override
  String get fillScanQr => 'Scan QR';

  @override
  String get openTextFile => 'Open a text file';

  @override
  String couldNotReadFile(String error) {
    return 'Could not read the file: $error';
  }

  @override
  String fieldRequired(String label) {
    return '$label *';
  }

  @override
  String formNoEditor(String protocol) {
    return 'No editor for $protocol';
  }

  @override
  String get shareScanHint =>
      'Scan it with Omnitun on another device, or copy the link.';

  @override
  String get shareCopy => 'Copy link';

  @override
  String get shareCopied => 'Link copied';

  @override
  String get shareWarning =>
      'The link contains this server\'s password or key: only share it with people who may use the server.';

  @override
  String shareFailed(String reason) {
    return 'This server cannot be shared as a link: $reason';
  }

  @override
  String get shareTooLong => 'Too long for a QR code: copy the link';

  @override
  String get obTitle1 => 'Private and simple';

  @override
  String get obText1 =>
      'Omnitun connects you through VPN, proxy and Tor protocols from one calm screen. Pick a place, press Connect.';

  @override
  String get obTitle2 => 'Add your servers';

  @override
  String get obText2 =>
      'Import a share link, a QR code, a config file or a subscription from your provider. You can also add a server by hand.';

  @override
  String get obTitle3Android => 'Android will ask once';

  @override
  String get obText3Android =>
      'The first time you connect, Android asks to allow a VPN connection. Omnitun can only protect your traffic after you accept. While connected, a notification shows the speed and a Disconnect button.';

  @override
  String get obTitle3Desktop => 'Ready when you are';

  @override
  String get obText3Desktop =>
      'Omnitun starts in proxy mode, which needs no special rights. Turn on VPN mode in Settings to capture all traffic.';

  @override
  String get obAddFirst => 'Add my first server';

  @override
  String get bypassTitle => 'Bypass lists';

  @override
  String get bypassAddTooltip => 'Add a list';

  @override
  String get bypassFromClipboard => 'Paste from the clipboard';

  @override
  String get bypassFromFile => 'Open a .txt file';

  @override
  String get bypassFromUrl => 'Download from a link';

  @override
  String get bypassType => 'Type it';

  @override
  String get bypassClipboardEmpty => 'The clipboard is empty';

  @override
  String get bypassPickFile => 'Open a bypass list (.txt)';

  @override
  String bypassDownloadFailed(String error) {
    return 'Could not download the list: $error';
  }

  @override
  String get bypassIntro =>
      'Websites and addresses that should not use the VPN, for example your bank or local services. One entry per line: example.com (and its subdomains), full:www.example.com, keyword:bank, regexp:…, 203.0.113.7 or 10.0.0.0/8. Lines starting with # are ignored.';

  @override
  String get bypassNoLists => 'No lists yet.';

  @override
  String get bypassTypeList => 'Type a list';

  @override
  String entriesCount(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count entries',
      one: '1 entry',
    );
    return '$_temp0';
  }

  @override
  String get bypassOnlyTitle => 'Only the listed destinations use the VPN';

  @override
  String get bypassOnlyOn => 'Everything else goes direct, outside the VPN.';

  @override
  String get bypassOnlyOff =>
      'Off: the listed destinations go direct and everything else uses the VPN.';

  @override
  String get bypassNothing => 'Nothing is bypassed until a list is on.';

  @override
  String bypassInUse(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count entries are in use',
      one: '1 entry is in use',
    );
    return '$_temp0. Changes apply the next time you connect.';
  }

  @override
  String get bypassHttpLink => 'Enter an http(s) link';

  @override
  String bypassServerAnswered(int code) {
    return 'the server answered $code';
  }

  @override
  String get bypassTooLarge => 'the list is larger than 5 MB';

  @override
  String get bypassDownloadTitle => 'Download a list';

  @override
  String get bypassNewList => 'New list';

  @override
  String get bypassEditList => 'Edit list';

  @override
  String get bypassName => 'Name';

  @override
  String get bypassEntriesLabel => 'Entries, one per line';

  @override
  String get bypassNoUsable => 'No usable entries yet.';

  @override
  String bypassSummary(String entries, String names, String ranges) {
    return '$entries · $names, $ranges.';
  }

  @override
  String bypassNames(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count names',
      one: '1 name',
    );
    return '$_temp0';
  }

  @override
  String bypassRanges(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count addresses or ranges',
      one: '1 address or range',
    );
    return '$_temp0';
  }

  @override
  String bypassIgnored(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count lines will be ignored:',
      one: '1 line will be ignored:',
    );
    return '$_temp0';
  }

  @override
  String bypassMoreIssues(int count) {
    return '… and $count more';
  }

  @override
  String get bypassReconnect => 'Reconnect to use the new list';

  @override
  String get bypassDefaultName => 'My list';

  @override
  String get bypassPastedName => 'Pasted list';

  @override
  String get settingsTitle => 'Settings';

  @override
  String get settingsServers => 'Servers';

  @override
  String get manageServers => 'Manage servers';

  @override
  String get manageServersHint => 'Import, subscriptions, order and removal';

  @override
  String get settingsConnection => 'Connection';

  @override
  String get vpnMode => 'VPN mode (all traffic)';

  @override
  String get vpnModeHint =>
      'Off = proxy only: local SOCKS/HTTP port, no VPN permission needed';

  @override
  String get systemProxy => 'Set system proxy';

  @override
  String get systemProxyHint =>
      'Proxy mode: point the OS proxy settings at Omnitun while connected';

  @override
  String get autoSwitch => 'Auto-switch on failure';

  @override
  String get autoSwitchHint =>
      'Try the next profile in the group when one stops working';

  @override
  String get bypassLan => 'Bypass local network';

  @override
  String get bypassLanHint =>
      'LAN devices and local addresses (192.168.x.x, 10.x.x.x, …) are reached directly, not through the VPN';

  @override
  String get bypassListsHint => 'Websites and addresses that skip the VPN';

  @override
  String bypassListsInUse(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count entries in use',
      one: '1 entry in use',
    );
    return '$_temp0';
  }

  @override
  String get alwaysOnTitle => 'Always-on VPN and kill switch';

  @override
  String get alwaysOnHint =>
      'Android reconnects at boot and can block all traffic while the VPN is down. In the VPN settings tap the gear next to Omnitun and turn on \"Always-on VPN\" and \"Block connections without VPN\". Connect once from this app first.';

  @override
  String get openVpnSettingsFallback =>
      'Open Android Settings → Network → VPN, then the gear next to Omnitun';

  @override
  String get splitTunneling => 'Split tunneling';

  @override
  String splitSummaryOnly(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: 'Only $count selected apps use the VPN',
      one: 'Only 1 selected app uses the VPN',
    );
    return '$_temp0';
  }

  @override
  String splitSummaryBypass(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count apps bypass the VPN',
      one: '1 app bypasses the VPN',
    );
    return '$_temp0';
  }

  @override
  String get splitSummaryOff => 'All apps use the VPN';

  @override
  String get remoteDns => 'Remote DNS';

  @override
  String get remoteDnsHelp =>
      'Resolver used through the tunnel. Examples: https://1.1.1.1/dns-query, tls://1.1.1.1, tcp://8.8.8.8, 9.9.9.9';

  @override
  String get settingsApp => 'App';

  @override
  String get appearance => 'Appearance';

  @override
  String get themeDark => 'Dark';

  @override
  String get themeLight => 'Light';

  @override
  String get themeSystem => 'Follow the system';

  @override
  String get language => 'Language';

  @override
  String get languageSystem => 'System default';

  @override
  String get stopOnClose => 'Stop VPN when app is closed';

  @override
  String aboutLine(String engine) {
    return 'MIT License · engine: $engine';
  }

  @override
  String get splitTitle => 'Split tunneling';

  @override
  String get splitAll => 'All apps';

  @override
  String get splitOnlySelected => 'Only selected';

  @override
  String get splitBypassSelected => 'Bypass selected';

  @override
  String get splitHelpOffMode =>
      'Every app uses the VPN. Choose a mode to pick apps.';

  @override
  String get splitHelpOnlyNone =>
      'No apps selected yet — until you pick some, all apps use the VPN.';

  @override
  String splitHelpOnly(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: 'Only the $count selected apps use the VPN.',
      one: 'Only the selected app uses the VPN.',
    );
    return '$_temp0';
  }

  @override
  String splitHelpBypass(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: 'The $count selected apps connect directly, without the VPN.',
      one: 'The selected app connects directly, without the VPN.',
    );
    return '$_temp0';
  }

  @override
  String get searchApps => 'Search apps';

  @override
  String get showSystemApps => 'Show system apps';

  @override
  String couldNotListApps(String error) {
    return 'Could not list apps: $error';
  }

  @override
  String get scanTitle => 'Scan QR code';

  @override
  String get scanTorch => 'Torch';

  @override
  String scanCameraUnavailable(String code) {
    return 'Camera unavailable: $code';
  }

  @override
  String get scanPoint => 'Point the camera at a config QR code';

  @override
  String get credTitle => 'Sign in';

  @override
  String credNeeds(String name) {
    return '$name needs a username and password.';
  }

  @override
  String get credUsername => 'Username';

  @override
  String get credPassword => 'Password';

  @override
  String get logsEmpty => 'No log entries yet.';

  @override
  String get profilesTitle => 'Manage servers';

  @override
  String get addSubscriptionTooltip => 'Add subscription';

  @override
  String get testLatencyTooltip => 'Test latency';

  @override
  String get profilesEmptyHint => 'No profiles. Tap Import to add one.';

  @override
  String get profilesDragHint =>
      'Long-press and drag to reorder. With auto-switch on, the rest of the group is tried next, in this order.';

  @override
  String get myProfiles => 'My profiles';

  @override
  String subRefreshed(String name, int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count profiles',
      one: '1 profile',
    );
    return '$name: $_temp0';
  }

  @override
  String get refreshTooltip => 'Refresh';

  @override
  String get subDeleteWithProfiles => 'Delete subscription and its profiles';

  @override
  String subUsageOf(String used, String total) {
    return '$used of $total';
  }

  @override
  String subUsed(String used) {
    return '$used used';
  }

  @override
  String get subExpired => 'expired';

  @override
  String get subExpiresToday => 'expires today';

  @override
  String subExpiresIn(int days) {
    String _temp0 = intl.Intl.pluralLogic(
      days,
      locale: localeName,
      other: 'expires in $days days',
      one: 'expires in 1 day',
    );
    return '$_temp0';
  }

  @override
  String subUpdated(String ago) {
    return 'updated $ago';
  }

  @override
  String get agoJustNow => 'just now';

  @override
  String agoMinutes(int count) {
    return '$count min ago';
  }

  @override
  String agoHours(int count) {
    return '$count h ago';
  }

  @override
  String agoDays(int count) {
    return '$count d ago';
  }

  @override
  String subRefreshFailed(String error) {
    return 'Last refresh failed: $error';
  }

  @override
  String get importTitle => 'Import config';

  @override
  String get importDescription =>
      'Paste share links (vless://, trojan://, ss://, hy2://, anytls://, ssh://, sstp://, ikev2://, …), a Clash or sing-box config, a subscription, a WireGuard .conf, an OpenVPN .ovpn or Tor bridge lines — or open a file or scan a QR code.';

  @override
  String get importScanQr => 'Scan QR';

  @override
  String get importOpenFile => 'Open file';

  @override
  String get importPaste => 'Paste';

  @override
  String get importSubscription => 'Subscription';

  @override
  String get importTor => 'Tor';

  @override
  String get importAddTor => 'Add Tor';

  @override
  String get importTorDirect => 'Tor (direct)';

  @override
  String get importTorSnowflake => 'Tor + Snowflake bridge';

  @override
  String get importTorObfs4 => 'Tor + obfs4 bridges';

  @override
  String importedProfiles(int added, int dupes) {
    String _temp0 = intl.Intl.pluralLogic(
      added,
      locale: localeName,
      other: 'Imported $added profiles',
      one: 'Imported 1 profile',
    );
    String _temp1 = intl.Intl.pluralLogic(
      dupes,
      locale: localeName,
      other: ', $dupes already present',
      zero: '',
    );
    return '$_temp0$_temp1';
  }

  @override
  String importNothing(String reason) {
    return 'Nothing imported: $reason';
  }

  @override
  String importPartial(int added, int failed) {
    return 'Imported $added, failed $failed';
  }

  @override
  String get subAddTitle => 'Add subscription';

  @override
  String get subEditTitle => 'Edit subscription';

  @override
  String get subUrlLabel => 'Subscription URL';

  @override
  String get subUrlHint =>
      'https://provider.example/sub/… or nordvpn://<token>';

  @override
  String get subNameLabel => 'Name (optional)';

  @override
  String get subNameHelp =>
      'Defaults to the provider\'s title or the host name';

  @override
  String get subAuthLabel => 'Authentication';

  @override
  String get authNone => 'None';

  @override
  String get authBasic => 'Username & password (Basic)';

  @override
  String get authBearer => 'Bearer token';

  @override
  String get authHeader => 'Custom header';

  @override
  String get authQuery => 'Token in URL';

  @override
  String get subUsername => 'Username';

  @override
  String get subPassword => 'Password';

  @override
  String get subHeaderName => 'Header name';

  @override
  String get subHeaderValue => 'Header value';

  @override
  String get subParamName => 'Parameter name';

  @override
  String get subToken => 'Token';

  @override
  String get subDownloadLabel => 'Download';

  @override
  String get subDownloadHelp =>
      'Some providers are blocked where you are; the VPN can still reach them';

  @override
  String get subViaDirect => 'Directly';

  @override
  String get subViaAuto =>
      'Directly, or through the VPN if the provider is blocked';

  @override
  String get subViaTunnel => 'Always through the VPN (connect first)';

  @override
  String get subErrUrl =>
      'Enter an http(s) subscription URL (or nordvpn://<access token>)';

  @override
  String get subErrHeader => 'Enter the header name';

  @override
  String get subErrToken => 'Enter the token';

  @override
  String get subAddAndDownload => 'Add and download';

  @override
  String get subSaveAndRefresh => 'Save and refresh';

  @override
  String subUpdatedToast(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count profiles',
      one: '1 profile',
    );
    return 'Subscription updated: $_temp0';
  }

  @override
  String get flName => 'Name';

  @override
  String get flServer => 'Server';

  @override
  String get flPort => 'Port';

  @override
  String get flPassword => 'Password';

  @override
  String get flUsername => 'Username';

  @override
  String get flUuid => 'UUID';

  @override
  String get flSecurity => 'Security';

  @override
  String get flFlow => 'Flow';

  @override
  String get flCipher => 'Cipher';

  @override
  String get flEncryption => 'Encryption';

  @override
  String get flAlterId => 'Alter id';

  @override
  String get flTlsServerName => 'TLS server name (SNI)';

  @override
  String get flTlsFingerprint => 'TLS fingerprint';

  @override
  String get flAlpn => 'ALPN';

  @override
  String get flSkipCertCheck => 'Skip certificate check';

  @override
  String get flPinnedCert => 'Pinned certificate (PEM)';

  @override
  String get flRealityPublicKey => 'Reality public key';

  @override
  String get flRealityShortId => 'Reality short id';

  @override
  String get flTransport => 'Transport';

  @override
  String get flHostHeader => 'Host header';

  @override
  String get flPath => 'Path';

  @override
  String get flGrpcService => 'gRPC service name';

  @override
  String get flPlugin => 'Plugin';

  @override
  String get flPluginOptions => 'Plugin options';

  @override
  String get flObfuscation => 'Obfuscation';

  @override
  String get flObfsPassword => 'Obfuscation password';

  @override
  String get flUploadMbps => 'Upload (Mbps)';

  @override
  String get flDownloadMbps => 'Download (Mbps)';

  @override
  String get flCongestion => 'Congestion control';

  @override
  String get flUdpRelay => 'UDP relay mode';

  @override
  String get flUseQuic => 'Use HTTP/3 (QUIC)';

  @override
  String get flTunnelAddress => 'Your tunnel address';

  @override
  String get flRoutedNetworks => 'Routed networks';

  @override
  String get flDnsServers => 'DNS servers';

  @override
  String get flMtu => 'MTU';

  @override
  String get flKeepalive => 'Keepalive (seconds)';

  @override
  String get flPrivateKey => 'Private key';

  @override
  String get flServerPublicKey => 'Server public key';

  @override
  String get flPresharedKey => 'Pre-shared key';

  @override
  String get flReservedBytes => 'Reserved bytes';

  @override
  String get flAuthentication => 'Authentication';

  @override
  String get flAuthDigest => 'Auth digest';

  @override
  String get flCaCert => 'CA certificate (PEM)';

  @override
  String get flClientCert => 'Client certificate (PEM)';

  @override
  String get flClientKey => 'Client key (PEM)';

  @override
  String get flControlWrap => 'Control channel protection';

  @override
  String get flControlWrapKey => 'Control channel key';

  @override
  String get flKeyDirection => 'Key direction';

  @override
  String get flRemoteCertType => 'Remote certificate type';

  @override
  String get flVerifyServerName => 'Verify server name';

  @override
  String get flYourIdentity => 'Your identity';

  @override
  String get flServerIdentity => 'Server identity';

  @override
  String get flServerCertTrust => 'Server certificate to trust (PEM)';

  @override
  String get flUseSystemIke => 'Use the phone\'s own IKEv2 client';

  @override
  String get flUseTcp => 'Use TCP';

  @override
  String get flHostKeys => 'Host keys';

  @override
  String get flPrivateKeyPem => 'Private key (PEM)';

  @override
  String get flKeyPassphrase => 'Key passphrase';

  @override
  String get flBridges => 'Bridges';

  @override
  String get flBridgeLines => 'Bridge lines';

  @override
  String get flJc => 'Jc (junk packet count)';

  @override
  String get flSignaturePacket => 'I1 (signature packet)';

  @override
  String get flControlledJunk => 'J1 (controlled junk)';

  @override
  String get hpAuthorizedKeys =>
      'authorized_keys format; empty accepts any key';

  @override
  String get hpIkeTcp =>
      'Carry IKE and ESP over TCP for networks that block UDP';

  @override
  String get hpDefaultsToServer => 'Defaults to the server address';

  @override
  String get hpTorBridges =>
      'Empty connects to Tor directly, or uses the bridge lines below';

  @override
  String get hpExampleAddress => 'For example 10.8.0.2/32';

  @override
  String get hpExamplePlugin => 'For example shadow-tls or v2ray-plugin';

  @override
  String get hpHostOrIp => 'Host name or IP address';

  @override
  String get hpNeededReality => 'Needed when Security is reality';

  @override
  String get hpOnlySelfSigned =>
      'Only for servers with a self-signed certificate you cannot pin';

  @override
  String get hpSafeSstp =>
      'Safe for SSTP: the password exchange proves the server';

  @override
  String get hpHysteriaObfs =>
      'salamander for Hysteria 2; the obfuscation password for Hysteria 1';

  @override
  String get hpTrustCert =>
      'Trust this certificate instead of the system\'s authorities';

  @override
  String get hpWhichBrowser => 'Which browser\'s TLS handshake to imitate';

  @override
  String get hpWarpReserved =>
      'Three numbers, as WARP configurations list them';

  @override
  String get hpOneBridgePerLine =>
      'One bridge per line, as in a torrc without \"Bridge \"';

  @override
  String get exitTitle => 'Show exit address and country';

  @override
  String get exitHint =>
      'After connecting, asks api.country.is, through the VPN, which address and country it sees. Off by default.';

  @override
  String exitIpLabel(String ip) {
    return 'IP $ip';
  }
}
