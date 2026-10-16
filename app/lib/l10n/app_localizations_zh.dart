// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Chinese (`zh`).
class AppLocalizationsZh extends AppLocalizations {
  AppLocalizationsZh([String locale = 'zh']) : super(locale);

  @override
  String get appName => 'Omnitun';

  @override
  String get actionSave => '保存';

  @override
  String get actionCancel => '取消';

  @override
  String get actionDelete => '删除';

  @override
  String get actionEdit => '编辑';

  @override
  String get actionReset => '重置';

  @override
  String get actionSkip => '跳过';

  @override
  String get actionNext => '下一步';

  @override
  String get actionImport => '导入';

  @override
  String get actionDownload => '下载';

  @override
  String get actionCopyAll => '全部复制';

  @override
  String get actionClear => '清除';

  @override
  String get tooltipMore => '更多';

  @override
  String get tooltipShow => '显示';

  @override
  String get tooltipHide => '隐藏';

  @override
  String get searchHint => '搜索';

  @override
  String get tooltipLogs => '日志';

  @override
  String get tooltipSettings => '设置';

  @override
  String get statusConnected => '已连接';

  @override
  String get statusConnecting => '正在连接…';

  @override
  String get statusDisconnecting => '正在断开…';

  @override
  String get statusError => '连接失败';

  @override
  String get statusNoInternet => '没有互联网连接';

  @override
  String get statusDisconnected => '未连接';

  @override
  String get stagePermission => '等待 VPN 权限';

  @override
  String get stageNetworkCheck => '正在检查互联网连接';

  @override
  String get stagePrepare => '正在准备配置';

  @override
  String get stageEngineStart => '正在启动引擎';

  @override
  String get stageTunnelUp => '正在打开隧道';

  @override
  String get stageHandshake => '正在联系服务器';

  @override
  String get stageConnected => '已连接';

  @override
  String get buttonConnect => '连接';

  @override
  String get buttonDisconnect => '断开连接';

  @override
  String get buttonCancel => '取消';

  @override
  String get locationLabel => '位置';

  @override
  String get noServers => '没有服务器';

  @override
  String get noServersHint => '添加一个即可开始';

  @override
  String get automatic => '自动';

  @override
  String get automaticHint => '最快的服务器，故障时自动切换';

  @override
  String get anyServerInCountry => '该国家的任意服务器';

  @override
  String get torNetwork => 'Tor 网络';

  @override
  String get emptyTitle => '还没有配置';

  @override
  String get emptyHint => '导入配置、链接或订阅。';

  @override
  String get importConfig => '导入配置';

  @override
  String failoverToast(String to, String from) {
    return '已切换到 $to — $from 不再响应';
  }

  @override
  String get unknownLocation => '未知位置';

  @override
  String errorFailed(String stage, String reason) {
    return '$stage失败：$reason';
  }

  @override
  String get errTimeout => '服务器没有及时响应';

  @override
  String get errRefused => '服务器拒绝了连接';

  @override
  String get errReset => '连接被网络重置';

  @override
  String get errNoRoute => '无法从此网络访问该服务器';

  @override
  String get errNetworkUnreachable => '网络不可达';

  @override
  String get errDns => '无法解析服务器名称';

  @override
  String get errTls => '服务器的安全证书被拒绝';

  @override
  String get errAuth => '用户名或密码被拒绝';

  @override
  String get errClosed => '服务器关闭了连接';

  @override
  String get errTorTimeout =>
      'Tor 没有及时连接完成。此网络可能屏蔽了 Tor；请使用网桥（Snowflake 或 obfs4）';

  @override
  String get errTorBridgeTimeout =>
      'Tor 没有及时连接完成。网桥可能被屏蔽或过载；请尝试其他类型的网桥（Snowflake 或 obfs4）';

  @override
  String get errTorBadBridge => 'Tor 网桥行无效（需要传输方式、地址和指纹）';

  @override
  String get errTorMissing => '此版本不包含 Tor 客户端';

  @override
  String get errPermission => '未授予 VPN 权限';

  @override
  String get errCoreLost => '与核心服务的连接已断开';

  @override
  String get selectLocation => '选择位置';

  @override
  String get testSpeedTooltip => '测试所有服务器的速度';

  @override
  String get addServersTooltip => '添加服务器';

  @override
  String get menuImport => '导入配置、链接或二维码';

  @override
  String get menuNewServer => '手动添加服务器';

  @override
  String get menuSubscription => '添加订阅';

  @override
  String get menuManage => '管理服务器和订阅';

  @override
  String serversCount(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count 台服务器',
    );
    return '$_temp0';
  }

  @override
  String get otherGroup => '其他';

  @override
  String get torGroup => 'Tor';

  @override
  String get noServersYet => '还没有服务器';

  @override
  String get importHintShort => '导入配置文件、链接或订阅。';

  @override
  String get showServers => '显示服务器';

  @override
  String get hideServers => '隐藏服务器';

  @override
  String get menuEditServer => '编辑';

  @override
  String get menuBridgeSettings => '网桥设置';

  @override
  String get menuShare => '分享链接或二维码';

  @override
  String get menuSetCountry => '设置国家';

  @override
  String get menuCredentials => '用户名和密码';

  @override
  String needsSignIn(String address) {
    return '$address · 需要登录';
  }

  @override
  String get countryOfServer => '此服务器所在国家';

  @override
  String get guessFromName => '根据名称判断';

  @override
  String get latencyNotTested => '未测试';

  @override
  String get latencyUnreachable => '无法连接';

  @override
  String get latencyTimeout => '超时';

  @override
  String latencyMs(int ms) {
    return '$ms 毫秒';
  }

  @override
  String get unknownCountry => '未知国家';

  @override
  String get addServerTitle => '添加服务器';

  @override
  String get addServerIntro => '选择协议。链接或二维码更快：请用导入。';

  @override
  String newServerTitle(String label) {
    return '新建 $label 服务器';
  }

  @override
  String get editServerTitle => '编辑服务器';

  @override
  String get torBridgesTitle => 'Tor 网桥';

  @override
  String get subscriptionEditNote => '此服务器来自订阅：订阅刷新时，你的更改会被替换。';

  @override
  String get moreOptions => '更多选项';

  @override
  String get countryRow => '国家';

  @override
  String get countryAuto => '根据名称和地址判断';

  @override
  String get alreadyInList => '此服务器已在列表中';

  @override
  String get choiceDefault => '默认';

  @override
  String get torBridgesNone => '无（直连，或使用下面的网桥行）';

  @override
  String get torBridgesSnowflake => 'Snowflake';

  @override
  String get torBridgesObfs4 => 'obfs4（内置网桥）';

  @override
  String get listHelper => '用逗号分隔多个值';

  @override
  String get fillPaste => '粘贴';

  @override
  String get fillFile => '文件';

  @override
  String get fillScanQr => '扫描二维码';

  @override
  String get openTextFile => '打开文本文件';

  @override
  String couldNotReadFile(String error) {
    return '无法读取文件：$error';
  }

  @override
  String fieldRequired(String label) {
    return '$label *';
  }

  @override
  String formNoEditor(String protocol) {
    return '没有 $protocol 的编辑器';
  }

  @override
  String get shareScanHint => '用 Omnitun 扫描，或复制链接。';

  @override
  String get shareCopy => '复制链接';

  @override
  String get shareCopied => '链接已复制';

  @override
  String get shareWarning => '包含服务器密码或密钥，请谨慎分享。';

  @override
  String shareFailed(String reason) {
    return '此服务器无法以链接形式分享：$reason';
  }

  @override
  String get shareTooLong => '内容过长，无法生成二维码：请复制链接';

  @override
  String get obTitle1 => '私密而简单';

  @override
  String get obText1 => 'VPN、代理和 Tor，一个简洁界面。选择位置，点按连接。';

  @override
  String get obTitle2 => '添加你的服务器';

  @override
  String get obText2 => '导入链接、二维码、配置文件或订阅，也可手动添加服务器。';

  @override
  String get obTitle3Android => 'Android 只会询问一次';

  @override
  String get obText3Android => 'Android 会请求允许 VPN 连接，请接受以获得保护。';

  @override
  String get obTitle3Desktop => '随时可用';

  @override
  String get obText3Desktop => '以代理模式启动，无需特殊权限。在设置中开启 VPN 模式可接管全部流量。';

  @override
  String get obAddFirst => '添加第一台服务器';

  @override
  String get bypassTitle => '绕过列表';

  @override
  String get bypassAddTooltip => '添加列表';

  @override
  String get bypassFromClipboard => '从剪贴板粘贴';

  @override
  String get bypassFromFile => '打开 .txt 文件';

  @override
  String get bypassFromUrl => '从链接下载';

  @override
  String get bypassType => '手动输入';

  @override
  String get bypassClipboardEmpty => '剪贴板为空';

  @override
  String get bypassPickFile => '打开绕过列表（.txt）';

  @override
  String bypassDownloadFailed(String error) {
    return '无法下载列表：$error';
  }

  @override
  String get bypassIntro =>
      '不走 VPN 的网站和地址。每行一条：example.com、full:www.example.com、keyword:bank、regexp:…、203.0.113.7 或 10.0.0.0/8。以 # 开头的行会被忽略。';

  @override
  String get bypassNoLists => '还没有列表。';

  @override
  String get bypassTypeList => '输入列表';

  @override
  String entriesCount(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count 条',
    );
    return '$_temp0';
  }

  @override
  String get bypassOnlyTitle => '仅列表内走 VPN';

  @override
  String get bypassOnlyOn => '其余直连。';

  @override
  String get bypassOnlyOff => '列表内直连，其余走 VPN。';

  @override
  String get bypassNothing => '未启用任何列表。';

  @override
  String bypassInUse(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '使用中 $count 条',
    );
    return '$_temp0。下次连接时生效。';
  }

  @override
  String get bypassHttpLink => '请输入 http(s) 链接';

  @override
  String bypassServerAnswered(int code) {
    return '服务器返回了 $code';
  }

  @override
  String get bypassTooLarge => '列表超过 5 MB';

  @override
  String get bypassDownloadTitle => '下载列表';

  @override
  String get bypassNewList => '新建列表';

  @override
  String get bypassEditList => '编辑列表';

  @override
  String get bypassName => '名称';

  @override
  String get bypassEntriesLabel => '条目，每行一条';

  @override
  String get bypassNoUsable => '还没有可用的条目。';

  @override
  String bypassSummary(String entries, String names, String ranges) {
    return '$entries · $names，$ranges。';
  }

  @override
  String bypassNames(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count 个域名',
    );
    return '$_temp0';
  }

  @override
  String bypassRanges(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count 个地址或地址段',
    );
    return '$_temp0';
  }

  @override
  String bypassIgnored(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '将忽略 $count 行：',
    );
    return '$_temp0';
  }

  @override
  String bypassMoreIssues(int count) {
    return '… 还有 $count 行';
  }

  @override
  String get bypassReconnect => '重新连接以使用新列表';

  @override
  String get bypassDefaultName => '我的列表';

  @override
  String get bypassPastedName => '粘贴的列表';

  @override
  String get settingsTitle => '设置';

  @override
  String get settingsServers => '服务器';

  @override
  String get manageServers => '管理服务器';

  @override
  String get manageServersHint => '服务器和订阅';

  @override
  String get settingsConnection => '连接';

  @override
  String get vpnMode => 'VPN 模式（所有流量）';

  @override
  String get vpnModeHint => '关闭：仅代理，无需 VPN 权限';

  @override
  String get systemProxy => '设置系统代理';

  @override
  String get systemProxyHint => '连接时设置系统代理';

  @override
  String get autoSwitch => '故障时自动切换';

  @override
  String get autoSwitchHint => '服务器失效时尝试下一个';

  @override
  String get bypassLan => '绕过本地网络';

  @override
  String get bypassLanHint => '直接访问本地设备（192.168.x.x、10.x.x.x）';

  @override
  String get bypassListsHint => '不走 VPN 的网站和地址';

  @override
  String bypassListsInUse(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '正在使用 $count 条',
    );
    return '$_temp0';
  }

  @override
  String get alwaysOnTitle => '始终开启的 VPN 与断网保护';

  @override
  String get alwaysOnHint =>
      '开机重连，VPN 断开时阻止流量。请在 Android VPN 设置中开启（Omnitun 旁的齿轮）。';

  @override
  String get openVpnSettingsFallback => '设置 → 网络 → VPN → Omnitun 旁的齿轮';

  @override
  String get splitTunneling => '分应用隧道';

  @override
  String splitSummaryOnly(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '仅 $count 个所选应用使用 VPN',
    );
    return '$_temp0';
  }

  @override
  String splitSummaryBypass(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count 个应用绕过 VPN',
    );
    return '$_temp0';
  }

  @override
  String get splitSummaryOff => '所有应用都使用 VPN';

  @override
  String get remoteDns => '远程 DNS';

  @override
  String get remoteDnsHelp =>
      '隧道内使用的 DNS，例如 https://1.1.1.1/dns-query 或 tls://1.1.1.1';

  @override
  String get settingsApp => '应用';

  @override
  String get appearance => '外观';

  @override
  String get themeDark => '深色';

  @override
  String get themeLight => '浅色';

  @override
  String get themeSystem => '跟随系统';

  @override
  String get language => '语言';

  @override
  String get languageSystem => '跟随系统';

  @override
  String get stopOnClose => '关闭应用时停止 VPN';

  @override
  String aboutLine(String engine) {
    return 'MIT 许可证 · 引擎：$engine';
  }

  @override
  String get splitTitle => '分应用隧道';

  @override
  String get splitAll => '所有应用';

  @override
  String get splitOnlySelected => '仅所选';

  @override
  String get splitBypassSelected => '所选绕过';

  @override
  String get splitHelpOffMode => '所有应用都走 VPN。';

  @override
  String get splitHelpOnlyNone => '未选应用：全部走 VPN。';

  @override
  String splitHelpOnly(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '仅 $count 个应用走 VPN。',
    );
    return '$_temp0';
  }

  @override
  String splitHelpBypass(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count 个应用直连。',
    );
    return '$_temp0';
  }

  @override
  String get searchApps => '搜索应用';

  @override
  String get showSystemApps => '显示系统应用';

  @override
  String couldNotListApps(String error) {
    return '无法列出应用：$error';
  }

  @override
  String get scanTitle => '扫描二维码';

  @override
  String get scanTorch => '手电筒';

  @override
  String scanCameraUnavailable(String code) {
    return '相机不可用：$code';
  }

  @override
  String get scanPoint => '将相机对准配置二维码';

  @override
  String get credTitle => '登录';

  @override
  String credNeeds(String name) {
    return '$name 需要用户名和密码。';
  }

  @override
  String get credUsername => '用户名';

  @override
  String get credPassword => '密码';

  @override
  String get logsEmpty => '还没有日志。';

  @override
  String get profilesTitle => '管理服务器';

  @override
  String get addSubscriptionTooltip => '添加订阅';

  @override
  String get testLatencyTooltip => '测试延迟';

  @override
  String get profilesEmptyHint => '没有配置。点击“导入”添加。';

  @override
  String get profilesDragHint => '长按拖动排序。故障切换按此顺序。';

  @override
  String get myProfiles => '我的配置';

  @override
  String subRefreshed(String name, int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count 个配置',
    );
    return '$name：$_temp0';
  }

  @override
  String get refreshTooltip => '刷新';

  @override
  String get subDeleteWithProfiles => '删除订阅及其配置';

  @override
  String subUsageOf(String used, String total) {
    return '$used / $total';
  }

  @override
  String subUsed(String used) {
    return '已用 $used';
  }

  @override
  String get subExpired => '已过期';

  @override
  String get subExpiresToday => '今天到期';

  @override
  String subExpiresIn(int days) {
    String _temp0 = intl.Intl.pluralLogic(
      days,
      locale: localeName,
      other: '$days 天后到期',
    );
    return '$_temp0';
  }

  @override
  String subUpdated(String ago) {
    return '$ago更新';
  }

  @override
  String get agoJustNow => '刚刚';

  @override
  String agoMinutes(int count) {
    return '$count 分钟前';
  }

  @override
  String agoHours(int count) {
    return '$count 小时前';
  }

  @override
  String agoDays(int count) {
    return '$count 天前';
  }

  @override
  String subRefreshFailed(String error) {
    return '上次刷新失败：$error';
  }

  @override
  String get importTitle => '导入配置';

  @override
  String get importDescription => '粘贴链接、配置或订阅，或打开文件、扫描二维码。';

  @override
  String get importScanQr => '扫描二维码';

  @override
  String get importOpenFile => '打开文件';

  @override
  String get importPaste => '粘贴';

  @override
  String get importSubscription => '订阅';

  @override
  String get importTor => 'Tor';

  @override
  String get importAddTor => '添加 Tor';

  @override
  String get importTorDirect => 'Tor（直连）';

  @override
  String get importTorSnowflake => 'Tor + Snowflake 网桥';

  @override
  String get importTorObfs4 => 'Tor + obfs4 网桥';

  @override
  String importedProfiles(int added, int dupes) {
    String _temp0 = intl.Intl.pluralLogic(
      added,
      locale: localeName,
      other: '$added 个配置',
    );
    String _temp1 = intl.Intl.pluralLogic(
      dupes,
      locale: localeName,
      other: '，其中 $dupes 个已存在',
      zero: '',
    );
    return '已导入 $_temp0$_temp1';
  }

  @override
  String importNothing(String reason) {
    return '没有导入任何内容：$reason';
  }

  @override
  String importPartial(int added, int failed) {
    return '已导入 $added 个，失败 $failed 个';
  }

  @override
  String get subAddTitle => '添加订阅';

  @override
  String get subEditTitle => '编辑订阅';

  @override
  String get subUrlLabel => '订阅链接';

  @override
  String get subUrlHint => 'https://provider.example/sub/… 或 nordvpn://<令牌>';

  @override
  String get subNameLabel => '名称（可选）';

  @override
  String get subNameHelp => '默认使用服务商的标题或主机名';

  @override
  String get subAuthLabel => '身份验证';

  @override
  String get authNone => '无';

  @override
  String get authBasic => '用户名和密码（Basic）';

  @override
  String get authBearer => 'Bearer 令牌';

  @override
  String get authHeader => '自定义请求头';

  @override
  String get authQuery => '链接中的令牌';

  @override
  String get subUsername => '用户名';

  @override
  String get subPassword => '密码';

  @override
  String get subHeaderName => '请求头名称';

  @override
  String get subHeaderValue => '请求头的值';

  @override
  String get subParamName => '参数名称';

  @override
  String get subToken => '令牌';

  @override
  String get subDownloadLabel => '下载方式';

  @override
  String get subDownloadHelp => '提供商被屏蔽？VPN 仍可获取';

  @override
  String get subViaDirect => '直接下载';

  @override
  String get subViaAuto => '直连，被屏蔽时走 VPN';

  @override
  String get subViaTunnel => '始终通过 VPN（需先连接）';

  @override
  String get subErrUrl => '请输入 http(s) 订阅链接（或 nordvpn://<访问令牌>）';

  @override
  String get subErrHeader => '请输入请求头名称';

  @override
  String get subErrToken => '请输入令牌';

  @override
  String get subAddAndDownload => '添加并下载';

  @override
  String get subSaveAndRefresh => '保存并刷新';

  @override
  String subUpdatedToast(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count 个配置',
    );
    return '订阅已更新：$_temp0';
  }

  @override
  String get flName => '名称';

  @override
  String get flServer => '服务器';

  @override
  String get flPort => '端口';

  @override
  String get flPassword => '密码';

  @override
  String get flUsername => '用户名';

  @override
  String get flUuid => 'UUID';

  @override
  String get flSecurity => '安全性';

  @override
  String get flFlow => 'Flow';

  @override
  String get flCipher => '加密算法';

  @override
  String get flEncryption => '加密方式';

  @override
  String get flAlterId => 'Alter id';

  @override
  String get flTlsServerName => 'TLS 服务器名称 (SNI)';

  @override
  String get flTlsFingerprint => 'TLS 指纹';

  @override
  String get flAlpn => 'ALPN';

  @override
  String get flSkipCertCheck => '跳过证书检查';

  @override
  String get flPinnedCert => '固定证书 (PEM)';

  @override
  String get flRealityPublicKey => 'Reality 公钥';

  @override
  String get flRealityShortId => 'Reality 短 ID';

  @override
  String get flTransport => '传输方式';

  @override
  String get flHostHeader => 'Host 请求头';

  @override
  String get flPath => '路径';

  @override
  String get flGrpcService => 'gRPC 服务名';

  @override
  String get flPlugin => '插件';

  @override
  String get flPluginOptions => '插件选项';

  @override
  String get flObfuscation => '混淆';

  @override
  String get flObfsPassword => '混淆密码';

  @override
  String get flUploadMbps => '上传 (Mbps)';

  @override
  String get flDownloadMbps => '下载 (Mbps)';

  @override
  String get flCongestion => '拥塞控制';

  @override
  String get flUdpRelay => 'UDP 中继模式';

  @override
  String get flUseQuic => '使用 HTTP/3 (QUIC)';

  @override
  String get flTunnelAddress => '你的隧道地址';

  @override
  String get flRoutedNetworks => '路由的网络';

  @override
  String get flDnsServers => 'DNS 服务器';

  @override
  String get flMtu => 'MTU';

  @override
  String get flKeepalive => 'Keepalive（秒）';

  @override
  String get flPrivateKey => '私钥';

  @override
  String get flServerPublicKey => '服务器公钥';

  @override
  String get flPresharedKey => '预共享密钥';

  @override
  String get flReservedBytes => '保留字节';

  @override
  String get flAuthentication => '身份验证';

  @override
  String get flAuthDigest => '验证摘要';

  @override
  String get flCaCert => 'CA 证书 (PEM)';

  @override
  String get flClientCert => '客户端证书 (PEM)';

  @override
  String get flClientKey => '客户端密钥 (PEM)';

  @override
  String get flControlWrap => '控制通道保护';

  @override
  String get flControlWrapKey => '控制通道密钥';

  @override
  String get flKeyDirection => '密钥方向';

  @override
  String get flRemoteCertType => '远端证书类型';

  @override
  String get flVerifyServerName => '验证服务器名称';

  @override
  String get flYourIdentity => '你的标识';

  @override
  String get flServerIdentity => '服务器标识';

  @override
  String get flServerCertTrust => '要信任的服务器证书 (PEM)';

  @override
  String get flUseSystemIke => '使用手机自带的 IKEv2 客户端';

  @override
  String get flUseTcp => '使用 TCP';

  @override
  String get flHostKeys => '主机密钥';

  @override
  String get flPrivateKeyPem => '私钥 (PEM)';

  @override
  String get flKeyPassphrase => '密钥口令';

  @override
  String get flBridges => '网桥';

  @override
  String get flBridgeLines => '网桥行';

  @override
  String get flJc => 'Jc（垃圾包数量）';

  @override
  String get flSignaturePacket => 'I1（签名包）';

  @override
  String get flControlledJunk => 'J1（受控垃圾包）';

  @override
  String get hpAuthorizedKeys => 'authorized_keys 格式；留空则接受任何密钥';

  @override
  String get hpIkeTcp => '通过 TCP 承载 IKE 和 ESP，适用于屏蔽 UDP 的网络';

  @override
  String get hpDefaultsToServer => '默认使用服务器地址';

  @override
  String get hpTorBridges => '留空则直连 Tor，或使用下面的网桥行';

  @override
  String get hpExampleAddress => '例如 10.8.0.2/32';

  @override
  String get hpExamplePlugin => '例如 shadow-tls 或 v2ray-plugin';

  @override
  String get hpHostOrIp => '主机名或 IP 地址';

  @override
  String get hpNeededReality => '当安全性为 reality 时需要';

  @override
  String get hpOnlySelfSigned => '仅适用于使用自签名证书且无法固定的服务器';

  @override
  String get hpSafeSstp => '对 SSTP 是安全的：密码交换可证明服务器身份';

  @override
  String get hpHysteriaObfs => 'Hysteria 2 填 salamander；Hysteria 1 填混淆密码';

  @override
  String get hpTrustCert => '信任此证书，而不是系统的证书颁发机构';

  @override
  String get hpWhichBrowser => '模仿哪个浏览器的 TLS 握手';

  @override
  String get hpWarpReserved => '三个数字，与 WARP 配置中的写法一致';

  @override
  String get hpOneBridgePerLine => '每行一个网桥，与 torrc 相同但不带“Bridge ”';

  @override
  String get exitTitle => '显示出口地址和国家';

  @override
  String get exitHint => '向 api.country.is 查询出口地址和国家。默认关闭。';

  @override
  String exitIpLabel(String ip) {
    return 'IP $ip';
  }

  @override
  String connectedTo(String place) {
    return '已连接到 $place';
  }

  @override
  String get noticesTitle => '开源许可';

  @override
  String get noticesIntro => 'Omnitun 采用 MIT 许可。它包含下列软件，各自适用其自己的许可。';

  @override
  String noticesMore(int count) {
    return '另外 $count 个';
  }

  @override
  String noticesPackages(int count) {
    return '$count 个软件包';
  }

  @override
  String get noticesLoadFailed => '无法读取许可列表。';
}
