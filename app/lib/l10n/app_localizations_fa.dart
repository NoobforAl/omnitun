// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Persian (`fa`).
class AppLocalizationsFa extends AppLocalizations {
  AppLocalizationsFa([String locale = 'fa']) : super(locale);

  @override
  String get appName => 'Omnitun';

  @override
  String get actionSave => 'ذخیره';

  @override
  String get actionCancel => 'لغو';

  @override
  String get actionDelete => 'حذف';

  @override
  String get actionEdit => 'ویرایش';

  @override
  String get actionReset => 'بازنشانی';

  @override
  String get actionSkip => 'رد شدن';

  @override
  String get actionNext => 'بعدی';

  @override
  String get actionImport => 'وارد کردن';

  @override
  String get actionDownload => 'دانلود';

  @override
  String get actionCopyAll => 'کپی همه';

  @override
  String get actionClear => 'پاک کردن';

  @override
  String get tooltipMore => 'بیشتر';

  @override
  String get tooltipShow => 'نمایش';

  @override
  String get tooltipHide => 'پنهان کردن';

  @override
  String get searchHint => 'جستجو';

  @override
  String get tooltipLogs => 'گزارش‌ها';

  @override
  String get tooltipSettings => 'تنظیمات';

  @override
  String get statusConnected => 'متصل است';

  @override
  String get statusConnecting => 'در حال اتصال…';

  @override
  String get statusDisconnecting => 'در حال قطع اتصال…';

  @override
  String get statusError => 'اتصال ناموفق بود';

  @override
  String get statusNoInternet => 'اینترنت در دسترس نیست';

  @override
  String get statusDisconnected => 'متصل نیست';

  @override
  String get stagePermission => 'در انتظار مجوز VPN';

  @override
  String get stageNetworkCheck => 'بررسی اتصال اینترنت';

  @override
  String get stagePrepare => 'آماده‌سازی پیکربندی';

  @override
  String get stageEngineStart => 'راه‌اندازی موتور';

  @override
  String get stageTunnelUp => 'باز کردن تونل';

  @override
  String get stageHandshake => 'برقراری ارتباط با سرور';

  @override
  String get stageConnected => 'متصل است';

  @override
  String get buttonConnect => 'اتصال';

  @override
  String get buttonDisconnect => 'قطع اتصال';

  @override
  String get buttonCancel => 'لغو';

  @override
  String get locationLabel => 'مکان';

  @override
  String get noServers => 'سروری وجود ندارد';

  @override
  String get noServersHint => 'برای شروع یکی اضافه کنید';

  @override
  String get automatic => 'خودکار';

  @override
  String get automaticHint => 'سریع‌ترین سرور؛ در صورت خرابی عوض می‌شود';

  @override
  String get anyServerInCountry => 'هر سروری در این کشور';

  @override
  String get torNetwork => 'شبکه Tor';

  @override
  String get emptyTitle => 'هنوز پروفایلی نیست';

  @override
  String get emptyHint => 'یک پیکربندی، لینک یا اشتراک وارد کنید.';

  @override
  String get importConfig => 'وارد کردن پیکربندی';

  @override
  String failoverToast(String to, String from) {
    return 'به $to تغییر کرد — $from پاسخ نمی‌داد';
  }

  @override
  String get unknownLocation => 'مکان نامشخص';

  @override
  String errorFailed(String stage, String reason) {
    return '$stage ناموفق بود: $reason';
  }

  @override
  String get errTimeout => 'سرور به‌موقع پاسخ نداد';

  @override
  String get errRefused => 'سرور اتصال را نپذیرفت';

  @override
  String get errReset => 'اتصال توسط شبکه قطع شد';

  @override
  String get errNoRoute => 'از این شبکه نمی‌توان به سرور رسید';

  @override
  String get errNetworkUnreachable => 'شبکه در دسترس نیست';

  @override
  String get errDns => 'نام سرور پیدا نشد';

  @override
  String get errTls => 'گواهی امنیتی سرور پذیرفته نشد';

  @override
  String get errAuth => 'نام کاربری یا گذرواژه پذیرفته نشد';

  @override
  String get errClosed => 'سرور اتصال را بست';

  @override
  String get errTorTimeout =>
      'Tor به‌موقع وصل نشد. ممکن است این شبکه Tor را مسدود کرده باشد؛ از پل (Snowflake یا obfs4) استفاده کنید';

  @override
  String get errTorBridgeTimeout =>
      'Tor به‌موقع وصل نشد. ممکن است پل مسدود یا شلوغ باشد؛ نوع دیگری از پل (Snowflake یا obfs4) را امتحان کنید';

  @override
  String get errTorBadBridge =>
      'خط پل Tor معتبر نیست (باید نوع انتقال، نشانی و اثر انگشت داشته باشد)';

  @override
  String get errTorMissing => 'این نسخه شامل برنامه Tor نیست';

  @override
  String get errPermission => 'مجوز VPN داده نشد';

  @override
  String get errCoreLost => 'ارتباط با سرویس هسته قطع شد';

  @override
  String get selectLocation => 'انتخاب مکان';

  @override
  String get testSpeedTooltip => 'آزمایش سرعت همه سرورها';

  @override
  String get addServersTooltip => 'افزودن سرور';

  @override
  String get menuImport => 'وارد کردن پیکربندی، لینک یا QR';

  @override
  String get menuNewServer => 'افزودن دستی سرور';

  @override
  String get menuSubscription => 'افزودن اشتراک';

  @override
  String get menuManage => 'مدیریت سرورها و اشتراک‌ها';

  @override
  String serversCount(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count سرور',
      one: '1 سرور',
    );
    return '$_temp0';
  }

  @override
  String get otherGroup => 'سایر';

  @override
  String get torGroup => 'Tor';

  @override
  String get noServersYet => 'هنوز سروری نیست';

  @override
  String get importHintShort => 'یک فایل پیکربندی، لینک یا اشتراک وارد کنید.';

  @override
  String get showServers => 'نمایش سرورها';

  @override
  String get hideServers => 'پنهان کردن سرورها';

  @override
  String get menuEditServer => 'ویرایش';

  @override
  String get menuBridgeSettings => 'تنظیمات پل';

  @override
  String get menuShare => 'اشتراک‌گذاری لینک یا QR';

  @override
  String get menuSetCountry => 'تعیین کشور';

  @override
  String get menuCredentials => 'نام کاربری و گذرواژه';

  @override
  String needsSignIn(String address) {
    return '$address · نیاز به ورود';
  }

  @override
  String get countryOfServer => 'کشور این سرور';

  @override
  String get guessFromName => 'تشخیص از روی نام';

  @override
  String get latencyNotTested => 'آزمایش نشده';

  @override
  String get latencyUnreachable => 'در دسترس نیست';

  @override
  String get latencyTimeout => 'زمان‌بر';

  @override
  String latencyMs(int ms) {
    return '$ms میلی‌ثانیه';
  }

  @override
  String get unknownCountry => 'کشور نامشخص';

  @override
  String get addServerTitle => 'افزودن سرور';

  @override
  String get addServerIntro =>
      'پروتکل را انتخاب کنید. لینک یا QR سریع‌تر است: از «وارد کردن» استفاده کنید.';

  @override
  String newServerTitle(String label) {
    return 'سرور $label جدید';
  }

  @override
  String get editServerTitle => 'ویرایش سرور';

  @override
  String get torBridgesTitle => 'پل‌های Tor';

  @override
  String get subscriptionEditNote =>
      'این سرور از یک اشتراک آمده است: با به‌روزرسانی اشتراک، تغییرات شما جایگزین می‌شود.';

  @override
  String get moreOptions => 'گزینه‌های بیشتر';

  @override
  String get countryRow => 'کشور';

  @override
  String get countryAuto => 'از روی نام و نشانی تشخیص داده می‌شود';

  @override
  String get alreadyInList => 'این سرور از قبل در فهرست است';

  @override
  String get choiceDefault => 'پیش‌فرض';

  @override
  String get torBridgesNone => 'هیچ (مستقیم، یا خط‌های زیر)';

  @override
  String get torBridgesSnowflake => 'Snowflake';

  @override
  String get torBridgesObfs4 => 'obfs4 (پل‌های داخلی)';

  @override
  String get listHelper => 'مقدارها را با ویرگول جدا کنید';

  @override
  String get fillPaste => 'چسباندن';

  @override
  String get fillFile => 'فایل';

  @override
  String get fillScanQr => 'اسکن QR';

  @override
  String get openTextFile => 'باز کردن فایل متنی';

  @override
  String couldNotReadFile(String error) {
    return 'خواندن فایل ممکن نشد: $error';
  }

  @override
  String fieldRequired(String label) {
    return '$label *';
  }

  @override
  String formNoEditor(String protocol) {
    return 'ویرایشگری برای $protocol وجود ندارد';
  }

  @override
  String get shareScanHint => 'با Omnitun اسکن کنید یا لینک را کپی کنید.';

  @override
  String get shareCopy => 'کپی لینک';

  @override
  String get shareCopied => 'لینک کپی شد';

  @override
  String get shareWarning =>
      'شامل گذرواژه یا کلید سرور است. با احتیاط به اشتراک بگذارید.';

  @override
  String shareFailed(String reason) {
    return 'این سرور را نمی‌توان به‌صورت لینک به اشتراک گذاشت: $reason';
  }

  @override
  String get shareTooLong =>
      'برای کد QR بیش از حد طولانی است: لینک را کپی کنید';

  @override
  String get obTitle1 => 'خصوصی و ساده';

  @override
  String get obText1 =>
      'یک صفحه آرام برای VPN، پروکسی و Tor. مکانی انتخاب کنید و «اتصال» را بزنید.';

  @override
  String get obTitle2 => 'سرورهایتان را اضافه کنید';

  @override
  String get obText2 =>
      'لینک، کد QR، فایل پیکربندی یا اشتراک را وارد کنید. یا سرور را دستی اضافه کنید.';

  @override
  String get obTitle3Android => 'اندروید یک بار اجازه می‌پرسد';

  @override
  String get obText3Android =>
      'اندروید اجازه‌ی اتصال VPN را می‌پرسد. برای محافظت، آن را بپذیرید.';

  @override
  String get obTitle3Desktop => 'آماده‌ایم';

  @override
  String get obText3Desktop =>
      'در حالت پروکسی و بدون دسترسی ویژه شروع می‌شود. برای عبور همه ترافیک، حالت VPN را در تنظیمات روشن کنید.';

  @override
  String get obAddFirst => 'افزودن اولین سرور';

  @override
  String get bypassTitle => 'فهرست‌های بدون VPN';

  @override
  String get bypassAddTooltip => 'افزودن فهرست';

  @override
  String get bypassFromClipboard => 'چسباندن از کلیپ‌بورد';

  @override
  String get bypassFromFile => 'باز کردن فایل txt.';

  @override
  String get bypassFromUrl => 'دانلود از یک لینک';

  @override
  String get bypassType => 'تایپ کردن';

  @override
  String get bypassClipboardEmpty => 'کلیپ‌بورد خالی است';

  @override
  String get bypassPickFile => 'باز کردن فهرست (txt.)';

  @override
  String bypassDownloadFailed(String error) {
    return 'دانلود فهرست ممکن نشد: $error';
  }

  @override
  String get bypassIntro =>
      'سایت‌ها و نشانی‌هایی که از VPN رد می‌شوند. هر خط یک مورد: ‎example.com‎، ‎full:www.example.com‎، ‎keyword:bank‎، ‎regexp:…‎، ‎203.0.113.7‎ یا ‎10.0.0.0/8‎. خط‌های شروع‌شونده با # نادیده گرفته می‌شوند.';

  @override
  String get bypassNoLists => 'هنوز فهرستی نیست.';

  @override
  String get bypassTypeList => 'تایپ یک فهرست';

  @override
  String entriesCount(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count مورد',
      one: '1 مورد',
    );
    return '$_temp0';
  }

  @override
  String get bypassOnlyTitle => 'فقط موارد فهرست از VPN استفاده کنند';

  @override
  String get bypassOnlyOn => 'بقیه مستقیم می‌روند.';

  @override
  String get bypassOnlyOff => 'موارد فهرست مستقیم؛ بقیه از VPN.';

  @override
  String get bypassNothing => 'هیچ فهرستی روشن نیست.';

  @override
  String bypassInUse(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count مورد در استفاده',
      one: '1 مورد در استفاده',
    );
    return '$_temp0. در اتصال بعدی اعمال می‌شود.';
  }

  @override
  String get bypassHttpLink => 'یک لینک http(s) وارد کنید';

  @override
  String bypassServerAnswered(int code) {
    return 'سرور پاسخ $code داد';
  }

  @override
  String get bypassTooLarge => 'فهرست بزرگ‌تر از ۵ مگابایت است';

  @override
  String get bypassDownloadTitle => 'دانلود فهرست';

  @override
  String get bypassNewList => 'فهرست جدید';

  @override
  String get bypassEditList => 'ویرایش فهرست';

  @override
  String get bypassName => 'نام';

  @override
  String get bypassEntriesLabel => 'موردها، هر خط یکی';

  @override
  String get bypassNoUsable => 'هنوز موردِ قابل‌استفاده‌ای نیست.';

  @override
  String bypassSummary(String entries, String names, String ranges) {
    return '$entries · $names، $ranges.';
  }

  @override
  String bypassNames(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count نام',
      one: '1 نام',
    );
    return '$_temp0';
  }

  @override
  String bypassRanges(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count نشانی یا محدوده',
      one: '1 نشانی یا محدوده',
    );
    return '$_temp0';
  }

  @override
  String bypassIgnored(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count خط نادیده گرفته می‌شود:',
      one: '1 خط نادیده گرفته می‌شود:',
    );
    return '$_temp0';
  }

  @override
  String bypassMoreIssues(int count) {
    return '… و $count مورد دیگر';
  }

  @override
  String get bypassReconnect => 'برای استفاده از فهرست جدید دوباره وصل شوید';

  @override
  String get bypassDefaultName => 'فهرست من';

  @override
  String get bypassPastedName => 'فهرست چسبانده‌شده';

  @override
  String get settingsTitle => 'تنظیمات';

  @override
  String get settingsServers => 'سرورها';

  @override
  String get manageServers => 'مدیریت سرورها';

  @override
  String get manageServersHint => 'سرورها و اشتراک‌ها';

  @override
  String get settingsConnection => 'اتصال';

  @override
  String get vpnMode => 'حالت VPN (همه ترافیک)';

  @override
  String get vpnModeHint => 'خاموش: فقط پروکسی، بدون مجوز VPN';

  @override
  String get systemProxy => 'تنظیم پروکسی سیستم';

  @override
  String get systemProxyHint => 'تنظیم پروکسی سیستم هنگام اتصال';

  @override
  String get autoSwitch => 'تغییر خودکار هنگام خرابی';

  @override
  String get autoSwitchHint => 'اگر سروری خراب شد، سرور بعدی امتحان شود';

  @override
  String get bypassLan => 'عبور مستقیم شبکه محلی';

  @override
  String get bypassLanHint =>
      'دستگاه‌های محلی مستقیم در دسترس‌اند (‎192.168.x.x‎، ‎10.x.x.x‎)';

  @override
  String get bypassListsHint =>
      'وب‌سایت‌ها و نشانی‌هایی که از VPN عبور نمی‌کنند';

  @override
  String bypassListsInUse(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count مورد در حال استفاده',
      one: '1 مورد در حال استفاده',
    );
    return '$_temp0';
  }

  @override
  String get alwaysOnTitle => 'VPN همیشه روشن و کلید قطع اضطراری';

  @override
  String get alwaysOnHint =>
      'اتصال دوباره هنگام راه‌اندازی و مسدود کردن ترافیک در صورت قطع VPN. در تنظیمات VPN اندروید (چرخ‌دنده کنار Omnitun) روشن کنید.';

  @override
  String get openVpnSettingsFallback =>
      'تنظیمات ← شبکه ← VPN ← چرخ‌دنده کنار Omnitun';

  @override
  String get splitTunneling => 'تونل تفکیکی';

  @override
  String splitSummaryOnly(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: 'فقط $count برنامه‌ی انتخاب‌شده از VPN استفاده می‌کنند',
      one: 'فقط 1 برنامه‌ی انتخاب‌شده از VPN استفاده می‌کند',
    );
    return '$_temp0';
  }

  @override
  String splitSummaryBypass(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count برنامه بدون VPN هستند',
      one: '1 برنامه بدون VPN است',
    );
    return '$_temp0';
  }

  @override
  String get splitSummaryOff => 'همه برنامه‌ها از VPN استفاده می‌کنند';

  @override
  String get remoteDns => 'DNS راه دور';

  @override
  String get remoteDnsHelp =>
      'DNS مورد استفاده در تونل، مثلاً ‎https://1.1.1.1/dns-query‎ یا ‎tls://1.1.1.1‎';

  @override
  String get settingsApp => 'برنامه';

  @override
  String get appearance => 'ظاهر';

  @override
  String get themeDark => 'تیره';

  @override
  String get themeLight => 'روشن';

  @override
  String get themeSystem => 'مطابق سیستم';

  @override
  String get language => 'زبان';

  @override
  String get languageSystem => 'پیش‌فرض سیستم';

  @override
  String get stopOnClose => 'قطع VPN هنگام بستن برنامه';

  @override
  String aboutLine(String engine) {
    return 'مجوز MIT · موتور: $engine';
  }

  @override
  String get splitTitle => 'تونل تفکیکی';

  @override
  String get splitAll => 'همه برنامه‌ها';

  @override
  String get splitOnlySelected => 'فقط انتخاب‌شده‌ها';

  @override
  String get splitBypassSelected => 'بدون VPN برای انتخاب‌شده‌ها';

  @override
  String get splitHelpOffMode => 'همه برنامه‌ها از VPN استفاده می‌کنند.';

  @override
  String get splitHelpOnlyNone =>
      'برنامه‌ای انتخاب نشده: همه از VPN استفاده می‌کنند.';

  @override
  String splitHelpOnly(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: 'فقط $count برنامه از VPN استفاده می‌کنند.',
      one: 'فقط 1 برنامه از VPN استفاده می‌کند.',
    );
    return '$_temp0';
  }

  @override
  String splitHelpBypass(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count برنامه مستقیم وصل می‌شوند.',
      one: '1 برنامه مستقیم وصل می‌شود.',
    );
    return '$_temp0';
  }

  @override
  String get searchApps => 'جستجوی برنامه‌ها';

  @override
  String get showSystemApps => 'نمایش برنامه‌های سیستمی';

  @override
  String couldNotListApps(String error) {
    return 'فهرست برنامه‌ها ممکن نشد: $error';
  }

  @override
  String get scanTitle => 'اسکن کد QR';

  @override
  String get scanTorch => 'چراغ‌قوه';

  @override
  String scanCameraUnavailable(String code) {
    return 'دوربین در دسترس نیست: $code';
  }

  @override
  String get scanPoint => 'دوربین را روی کد QR پیکربندی بگیرید';

  @override
  String get credTitle => 'ورود';

  @override
  String credNeeds(String name) {
    return '$name به نام کاربری و گذرواژه نیاز دارد.';
  }

  @override
  String get credUsername => 'نام کاربری';

  @override
  String get credPassword => 'گذرواژه';

  @override
  String get logsEmpty => 'هنوز گزارشی ثبت نشده است.';

  @override
  String get profilesTitle => 'مدیریت سرورها';

  @override
  String get addSubscriptionTooltip => 'افزودن اشتراک';

  @override
  String get testLatencyTooltip => 'آزمایش تأخیر';

  @override
  String get profilesEmptyHint =>
      'پروفایلی نیست. برای افزودن، «وارد کردن» را بزنید.';

  @override
  String get profilesDragHint =>
      'برای تغییر ترتیب نگه دارید و بکشید. تعویض خودکار به همین ترتیب است.';

  @override
  String get myProfiles => 'پروفایل‌های من';

  @override
  String subRefreshed(String name, int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count پروفایل',
      one: '1 پروفایل',
    );
    return '$name: $_temp0';
  }

  @override
  String get refreshTooltip => 'به‌روزرسانی';

  @override
  String get subDeleteWithProfiles => 'حذف اشتراک و پروفایل‌هایش';

  @override
  String subUsageOf(String used, String total) {
    return '$used از $total';
  }

  @override
  String subUsed(String used) {
    return '$used مصرف شده';
  }

  @override
  String get subExpired => 'منقضی شده';

  @override
  String get subExpiresToday => 'امروز منقضی می‌شود';

  @override
  String subExpiresIn(int days) {
    String _temp0 = intl.Intl.pluralLogic(
      days,
      locale: localeName,
      other: '$days روز دیگر منقضی می‌شود',
      one: '1 روز دیگر منقضی می‌شود',
    );
    return '$_temp0';
  }

  @override
  String subUpdated(String ago) {
    return 'به‌روزرسانی: $ago';
  }

  @override
  String get agoJustNow => 'همین الان';

  @override
  String agoMinutes(int count) {
    return '$count دقیقه پیش';
  }

  @override
  String agoHours(int count) {
    return '$count ساعت پیش';
  }

  @override
  String agoDays(int count) {
    return '$count روز پیش';
  }

  @override
  String subRefreshFailed(String error) {
    return 'آخرین به‌روزرسانی ناموفق بود: $error';
  }

  @override
  String get importTitle => 'وارد کردن پیکربندی';

  @override
  String get importDescription =>
      'لینک، پیکربندی یا اشتراک را بچسبانید — یا فایلی باز کنید یا کد QR اسکن کنید.';

  @override
  String get importScanQr => 'اسکن QR';

  @override
  String get importOpenFile => 'باز کردن فایل';

  @override
  String get importPaste => 'چسباندن';

  @override
  String get importSubscription => 'اشتراک';

  @override
  String get importTor => 'Tor';

  @override
  String get importAddTor => 'افزودن Tor';

  @override
  String get importTorDirect => 'Tor (مستقیم)';

  @override
  String get importTorSnowflake => 'Tor + پل Snowflake';

  @override
  String get importTorObfs4 => 'Tor + پل‌های obfs4';

  @override
  String importedProfiles(int added, int dupes) {
    String _temp0 = intl.Intl.pluralLogic(
      added,
      locale: localeName,
      other: '$added پروفایل وارد شد',
      one: '1 پروفایل وارد شد',
    );
    String _temp1 = intl.Intl.pluralLogic(
      dupes,
      locale: localeName,
      other: '، $dupes مورد از قبل بود',
      zero: '',
    );
    return '$_temp0$_temp1';
  }

  @override
  String importNothing(String reason) {
    return 'چیزی وارد نشد: $reason';
  }

  @override
  String importPartial(int added, int failed) {
    return '$added مورد وارد شد، $failed مورد ناموفق بود';
  }

  @override
  String get subAddTitle => 'افزودن اشتراک';

  @override
  String get subEditTitle => 'ویرایش اشتراک';

  @override
  String get subUrlLabel => 'نشانی اشتراک';

  @override
  String get subUrlHint =>
      'https://provider.example/sub/… یا nordvpn://<token>';

  @override
  String get subNameLabel => 'نام (اختیاری)';

  @override
  String get subNameHelp => 'به‌طور پیش‌فرض عنوان ارائه‌دهنده یا نام میزبان';

  @override
  String get subAuthLabel => 'احراز هویت';

  @override
  String get authNone => 'هیچ';

  @override
  String get authBasic => 'نام کاربری و گذرواژه (Basic)';

  @override
  String get authBearer => 'توکن Bearer';

  @override
  String get authHeader => 'سرآیند سفارشی';

  @override
  String get authQuery => 'توکن در نشانی';

  @override
  String get subUsername => 'نام کاربری';

  @override
  String get subPassword => 'گذرواژه';

  @override
  String get subHeaderName => 'نام سرآیند';

  @override
  String get subHeaderValue => 'مقدار سرآیند';

  @override
  String get subParamName => 'نام پارامتر';

  @override
  String get subToken => 'توکن';

  @override
  String get subDownloadLabel => 'دانلود';

  @override
  String get subDownloadHelp => 'ارائه‌دهنده مسدود است؟ VPN باز هم می‌رسد';

  @override
  String get subViaDirect => 'مستقیم';

  @override
  String get subViaAuto => 'مستقیم، یا با VPN اگر مسدود بود';

  @override
  String get subViaTunnel => 'همیشه از طریق VPN (ابتدا وصل شوید)';

  @override
  String get subErrUrl =>
      'نشانی http(s) اشتراک (یا nordvpn://<توکن دسترسی>) را وارد کنید';

  @override
  String get subErrHeader => 'نام سرآیند را وارد کنید';

  @override
  String get subErrToken => 'توکن را وارد کنید';

  @override
  String get subAddAndDownload => 'افزودن و دانلود';

  @override
  String get subSaveAndRefresh => 'ذخیره و به‌روزرسانی';

  @override
  String subUpdatedToast(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count پروفایل',
      one: '1 پروفایل',
    );
    return 'اشتراک به‌روز شد: $_temp0';
  }

  @override
  String get flName => 'نام';

  @override
  String get flServer => 'سرور';

  @override
  String get flPort => 'پورت';

  @override
  String get flPassword => 'گذرواژه';

  @override
  String get flUsername => 'نام کاربری';

  @override
  String get flUuid => 'UUID';

  @override
  String get flSecurity => 'امنیت';

  @override
  String get flFlow => 'Flow';

  @override
  String get flCipher => 'رمز';

  @override
  String get flEncryption => 'رمزگذاری';

  @override
  String get flAlterId => 'Alter id';

  @override
  String get flTlsServerName => 'نام سرور TLS (SNI)';

  @override
  String get flTlsFingerprint => 'اثر انگشت TLS';

  @override
  String get flAlpn => 'ALPN';

  @override
  String get flSkipCertCheck => 'رد کردن بررسی گواهی';

  @override
  String get flPinnedCert => 'گواهی پین‌شده (PEM)';

  @override
  String get flRealityPublicKey => 'کلید عمومی Reality';

  @override
  String get flRealityShortId => 'شناسه کوتاه Reality';

  @override
  String get flTransport => 'انتقال';

  @override
  String get flHostHeader => 'سرآیند میزبان';

  @override
  String get flPath => 'مسیر';

  @override
  String get flGrpcService => 'نام سرویس gRPC';

  @override
  String get flPlugin => 'افزونه';

  @override
  String get flPluginOptions => 'گزینه‌های افزونه';

  @override
  String get flObfuscation => 'مخفی‌سازی';

  @override
  String get flObfsPassword => 'گذرواژه مخفی‌سازی';

  @override
  String get flUploadMbps => 'آپلود (مگابیت بر ثانیه)';

  @override
  String get flDownloadMbps => 'دانلود (مگابیت بر ثانیه)';

  @override
  String get flCongestion => 'کنترل ازدحام';

  @override
  String get flUdpRelay => 'حالت رله UDP';

  @override
  String get flUseQuic => 'استفاده از HTTP/3 (QUIC)';

  @override
  String get flTunnelAddress => 'نشانی تونل شما';

  @override
  String get flRoutedNetworks => 'شبکه‌های مسیردهی‌شده';

  @override
  String get flDnsServers => 'سرورهای DNS';

  @override
  String get flMtu => 'MTU';

  @override
  String get flKeepalive => 'Keepalive (ثانیه)';

  @override
  String get flPrivateKey => 'کلید خصوصی';

  @override
  String get flServerPublicKey => 'کلید عمومی سرور';

  @override
  String get flPresharedKey => 'کلید از پیش‌اشتراکی';

  @override
  String get flReservedBytes => 'بایت‌های رزرو';

  @override
  String get flAuthentication => 'احراز هویت';

  @override
  String get flAuthDigest => 'چکیده احراز هویت';

  @override
  String get flCaCert => 'گواهی CA (PEM)';

  @override
  String get flClientCert => 'گواهی کلاینت (PEM)';

  @override
  String get flClientKey => 'کلید کلاینت (PEM)';

  @override
  String get flControlWrap => 'محافظت کانال کنترل';

  @override
  String get flControlWrapKey => 'کلید کانال کنترل';

  @override
  String get flKeyDirection => 'جهت کلید';

  @override
  String get flRemoteCertType => 'نوع گواهی سرور';

  @override
  String get flVerifyServerName => 'بررسی نام سرور';

  @override
  String get flYourIdentity => 'هویت شما';

  @override
  String get flServerIdentity => 'هویت سرور';

  @override
  String get flServerCertTrust => 'گواهی سرور برای اعتماد (PEM)';

  @override
  String get flUseSystemIke => 'استفاده از کلاینت IKEv2 خودِ گوشی';

  @override
  String get flUseTcp => 'استفاده از TCP';

  @override
  String get flHostKeys => 'کلیدهای میزبان';

  @override
  String get flPrivateKeyPem => 'کلید خصوصی (PEM)';

  @override
  String get flKeyPassphrase => 'عبارت عبور کلید';

  @override
  String get flBridges => 'پل‌ها';

  @override
  String get flBridgeLines => 'خط‌های پل';

  @override
  String get flJc => 'Jc (تعداد بسته‌های زائد)';

  @override
  String get flSignaturePacket => 'I1 (بسته امضا)';

  @override
  String get flControlledJunk => 'J1 (بسته زائد کنترل‌شده)';

  @override
  String get hpAuthorizedKeys =>
      'قالب authorized_keys؛ خالی یعنی هر کلیدی پذیرفته می‌شود';

  @override
  String get hpIkeTcp =>
      'IKE و ESP را روی TCP عبور می‌دهد، برای شبکه‌هایی که UDP را مسدود می‌کنند';

  @override
  String get hpDefaultsToServer => 'به‌طور پیش‌فرض نشانی سرور';

  @override
  String get hpTorBridges =>
      'خالی یعنی اتصال مستقیم به Tor، یا استفاده از خط‌های پل زیر';

  @override
  String get hpExampleAddress => 'برای مثال 10.8.0.2/32';

  @override
  String get hpExamplePlugin => 'برای مثال shadow-tls یا v2ray-plugin';

  @override
  String get hpHostOrIp => 'نام میزبان یا نشانی IP';

  @override
  String get hpNeededReality => 'وقتی امنیت روی reality است لازم است';

  @override
  String get hpOnlySelfSigned =>
      'فقط برای سرورهایی با گواهی خودامضا که نمی‌توانید پین کنید';

  @override
  String get hpSafeSstp =>
      'برای SSTP ایمن است: تبادل گذرواژه سرور را اثبات می‌کند';

  @override
  String get hpHysteriaObfs =>
      'salamander برای Hysteria 2؛ گذرواژه‌ی مخفی‌سازی برای Hysteria 1';

  @override
  String get hpTrustCert => 'به‌جای مراجع سیستم، به این گواهی اعتماد کن';

  @override
  String get hpWhichBrowser => 'هندشیک TLS کدام مرورگر تقلید شود';

  @override
  String get hpWarpReserved =>
      'سه عدد، همان‌طور که در پیکربندی‌های WARP آمده است';

  @override
  String get hpOneBridgePerLine =>
      'در هر خط یک پل، مانند torrc ولی بدون «Bridge »';

  @override
  String get exitTitle => 'نمایش نشانی و کشور خروجی';

  @override
  String get exitHint =>
      'نشانی و کشور خروجی را از api.country.is بپرس. پیش‌فرض: خاموش.';

  @override
  String exitIpLabel(String ip) {
    return 'IP‎ $ip';
  }

  @override
  String connectedTo(String place) {
    return 'متصل به $place';
  }

  @override
  String get noticesTitle => 'مجوزهای متن‌باز';

  @override
  String get noticesIntro =>
      'Omnitun با مجوز MIT منتشر شده است. نرم‌افزارهای زیر هر کدام با مجوز خودشان در آن به کار رفته‌اند.';

  @override
  String noticesMore(int count) {
    return '+$count مورد دیگر';
  }

  @override
  String noticesPackages(int count) {
    return '$count بسته';
  }

  @override
  String get noticesLoadFailed => 'فهرست مجوزها خوانده نشد.';
}
