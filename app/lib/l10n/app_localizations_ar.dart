// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Arabic (`ar`).
class AppLocalizationsAr extends AppLocalizations {
  AppLocalizationsAr([String locale = 'ar']) : super(locale);

  @override
  String get appName => 'Omnitun';

  @override
  String get actionSave => 'حفظ';

  @override
  String get actionCancel => 'إلغاء';

  @override
  String get actionDelete => 'حذف';

  @override
  String get actionEdit => 'تعديل';

  @override
  String get actionReset => 'إعادة ضبط';

  @override
  String get actionSkip => 'تخطي';

  @override
  String get actionNext => 'التالي';

  @override
  String get actionImport => 'استيراد';

  @override
  String get actionDownload => 'تنزيل';

  @override
  String get actionCopyAll => 'نسخ الكل';

  @override
  String get actionClear => 'مسح';

  @override
  String get tooltipMore => 'المزيد';

  @override
  String get tooltipShow => 'إظهار';

  @override
  String get tooltipHide => 'إخفاء';

  @override
  String get searchHint => 'بحث';

  @override
  String get tooltipLogs => 'السجلات';

  @override
  String get tooltipSettings => 'الإعدادات';

  @override
  String get statusConnected => 'متصل';

  @override
  String get statusConnecting => 'جارٍ الاتصال…';

  @override
  String get statusDisconnecting => 'جارٍ قطع الاتصال…';

  @override
  String get statusError => 'فشل الاتصال';

  @override
  String get statusNoInternet => 'لا يوجد اتصال بالإنترنت';

  @override
  String get statusDisconnected => 'غير متصل';

  @override
  String get stagePermission => 'في انتظار إذن VPN';

  @override
  String get stageNetworkCheck => 'التحقق من اتصال الإنترنت';

  @override
  String get stagePrepare => 'تجهيز الإعدادات';

  @override
  String get stageEngineStart => 'تشغيل المحرك';

  @override
  String get stageTunnelUp => 'فتح النفق';

  @override
  String get stageHandshake => 'الاتصال بالخادم';

  @override
  String get stageConnected => 'متصل';

  @override
  String get buttonConnect => 'اتصال';

  @override
  String get buttonDisconnect => 'قطع الاتصال';

  @override
  String get buttonCancel => 'إلغاء';

  @override
  String get locationLabel => 'الموقع';

  @override
  String get noServers => 'لا توجد خوادم';

  @override
  String get noServersHint => 'أضف خادمًا للبدء';

  @override
  String get automatic => 'تلقائي';

  @override
  String get automaticHint => 'أسرع خادم، ويتبدّل عند العطل';

  @override
  String get anyServerInCountry => 'أي خادم في هذا البلد';

  @override
  String get torNetwork => 'شبكة Tor';

  @override
  String get emptyTitle => 'لا توجد ملفات تعريف بعد';

  @override
  String get emptyHint => 'استورد إعدادًا أو رابطًا أو اشتراكًا.';

  @override
  String get importConfig => 'استيراد الإعدادات';

  @override
  String failoverToast(String to, String from) {
    return 'تم التبديل إلى $to — توقف $from عن الاستجابة';
  }

  @override
  String get unknownLocation => 'موقع غير معروف';

  @override
  String errorFailed(String stage, String reason) {
    return 'فشل $stage: $reason';
  }

  @override
  String get errTimeout => 'لم يستجب الخادم في الوقت المناسب';

  @override
  String get errRefused => 'رفض الخادم الاتصال';

  @override
  String get errReset => 'أعادت الشبكة ضبط الاتصال';

  @override
  String get errNoRoute => 'لا يمكن الوصول إلى الخادم من هذه الشبكة';

  @override
  String get errNetworkUnreachable => 'الشبكة غير قابلة للوصول';

  @override
  String get errDns => 'تعذّر تحديد عنوان الخادم من اسمه';

  @override
  String get errTls => 'تم رفض شهادة أمان الخادم';

  @override
  String get errAuth => 'تم رفض اسم المستخدم أو كلمة المرور';

  @override
  String get errClosed => 'أغلق الخادم الاتصال';

  @override
  String get errTorTimeout =>
      'لم يكتمل اتصال Tor في الوقت المناسب. قد تحجب هذه الشبكة Tor؛ استخدم جسرًا (Snowflake أو obfs4)';

  @override
  String get errTorBridgeTimeout =>
      'لم يكتمل اتصال Tor في الوقت المناسب. قد يكون الجسر محجوبًا أو مزدحمًا؛ جرّب نوعًا آخر من الجسور (Snowflake أو obfs4)';

  @override
  String get errTorBadBridge =>
      'سطر جسر Tor غير صالح (يلزمه نوع النقل والعنوان والبصمة)';

  @override
  String get errTorMissing => 'لا يتضمن هذا الإصدار عميل Tor';

  @override
  String get errPermission => 'لم يتم منح إذن VPN';

  @override
  String get errCoreLost => 'انقطع الاتصال بخدمة النواة';

  @override
  String get selectLocation => 'اختيار الموقع';

  @override
  String get testSpeedTooltip => 'اختبار سرعة جميع الخوادم';

  @override
  String get addServersTooltip => 'إضافة خوادم';

  @override
  String get menuImport => 'استيراد إعدادات أو رابط أو رمز QR';

  @override
  String get menuNewServer => 'إضافة خادم يدويًا';

  @override
  String get menuSubscription => 'إضافة اشتراك';

  @override
  String get menuManage => 'إدارة الخوادم والاشتراكات';

  @override
  String serversCount(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count خادم',
      many: '$count خادمًا',
      few: '$count خوادم',
      two: 'خادمان',
      one: 'خادم واحد',
      zero: 'لا خوادم',
    );
    return '$_temp0';
  }

  @override
  String get otherGroup => 'أخرى';

  @override
  String get torGroup => 'Tor';

  @override
  String get noServersYet => 'لا توجد خوادم بعد';

  @override
  String get importHintShort => 'استورد ملف إعدادات أو رابطًا أو اشتراكًا.';

  @override
  String get showServers => 'إظهار الخوادم';

  @override
  String get hideServers => 'إخفاء الخوادم';

  @override
  String get menuEditServer => 'تعديل';

  @override
  String get menuBridgeSettings => 'إعدادات الجسور';

  @override
  String get menuShare => 'مشاركة الرابط أو رمز QR';

  @override
  String get menuSetCountry => 'تحديد البلد';

  @override
  String get menuCredentials => 'اسم المستخدم وكلمة المرور';

  @override
  String needsSignIn(String address) {
    return '$address · يتطلب تسجيل الدخول';
  }

  @override
  String get countryOfServer => 'بلد هذا الخادم';

  @override
  String get guessFromName => 'استنتاجه من الاسم';

  @override
  String get latencyNotTested => 'لم يُختبر';

  @override
  String get latencyUnreachable => 'غير قابل للوصول';

  @override
  String get latencyTimeout => 'انتهت المهلة';

  @override
  String latencyMs(int ms) {
    return '$ms مللي ثانية';
  }

  @override
  String get unknownCountry => 'بلد غير معروف';

  @override
  String get addServerTitle => 'إضافة خادم';

  @override
  String get addServerIntro =>
      'اختر البروتوكول. الرابط أو QR أسرع: استخدم استيراد.';

  @override
  String newServerTitle(String label) {
    return 'خادم $label جديد';
  }

  @override
  String get editServerTitle => 'تعديل الخادم';

  @override
  String get torBridgesTitle => 'جسور Tor';

  @override
  String get subscriptionEditNote =>
      'هذا الخادم من اشتراك: ستُستبدل تغييراتك عند تحديث الاشتراك.';

  @override
  String get moreOptions => 'خيارات إضافية';

  @override
  String get countryRow => 'البلد';

  @override
  String get countryAuto => 'يُستنتج من الاسم والعنوان';

  @override
  String get alreadyInList => 'هذا الخادم موجود بالفعل في القائمة';

  @override
  String get choiceDefault => 'افتراضي';

  @override
  String get torBridgesNone => 'بلا (اتصال مباشر، أو الأسطر أدناه)';

  @override
  String get torBridgesSnowflake => 'Snowflake';

  @override
  String get torBridgesObfs4 => 'obfs4 (جسور مدمجة)';

  @override
  String get listHelper => 'افصل القيم بفواصل';

  @override
  String get fillPaste => 'لصق';

  @override
  String get fillFile => 'ملف';

  @override
  String get fillScanQr => 'مسح QR';

  @override
  String get openTextFile => 'فتح ملف نصي';

  @override
  String couldNotReadFile(String error) {
    return 'تعذّرت قراءة الملف: $error';
  }

  @override
  String fieldRequired(String label) {
    return '$label *';
  }

  @override
  String formNoEditor(String protocol) {
    return 'لا يوجد محرر لـ $protocol';
  }

  @override
  String get shareScanHint => 'امسحه بـ Omnitun أو انسخ الرابط.';

  @override
  String get shareCopy => 'نسخ الرابط';

  @override
  String get shareCopied => 'تم نسخ الرابط';

  @override
  String get shareWarning =>
      'يحتوي على كلمة مرور الخادم أو مفتاحه. شاركه بحذر.';

  @override
  String shareFailed(String reason) {
    return 'لا يمكن مشاركة هذا الخادم كرابط: $reason';
  }

  @override
  String get shareTooLong => 'أطول من أن يُحوَّل إلى رمز QR: انسخ الرابط';

  @override
  String get obTitle1 => 'خصوصية وبساطة';

  @override
  String get obText1 =>
      'شاشة هادئة واحدة لـ VPN والوكيل وTor. اختر مكانًا واضغط اتصال.';

  @override
  String get obTitle2 => 'أضف خوادمك';

  @override
  String get obText2 =>
      'استورد رابطًا أو رمز QR أو ملف إعداد أو اشتراكًا، أو أضف خادمًا يدويًا.';

  @override
  String get obTitle3Android => 'سيسألك أندرويد مرة واحدة';

  @override
  String get obText3Android =>
      'سيطلب أندرويد السماح باتصال VPN. اقبل ليبدأ الحماية.';

  @override
  String get obTitle3Desktop => 'جاهز متى شئت';

  @override
  String get obText3Desktop =>
      'يبدأ بوضع الوكيل دون صلاحيات خاصة. فعّل وضع VPN من الإعدادات لالتقاط كل الحركة.';

  @override
  String get obAddFirst => 'إضافة أول خادم';

  @override
  String get bypassTitle => 'قوائم التجاوز';

  @override
  String get bypassAddTooltip => 'إضافة قائمة';

  @override
  String get bypassFromClipboard => 'لصق من الحافظة';

  @override
  String get bypassFromFile => 'فتح ملف .txt';

  @override
  String get bypassFromUrl => 'تنزيل من رابط';

  @override
  String get bypassType => 'كتابتها';

  @override
  String get bypassClipboardEmpty => 'الحافظة فارغة';

  @override
  String get bypassPickFile => 'فتح قائمة تجاوز (.txt)';

  @override
  String bypassDownloadFailed(String error) {
    return 'تعذّر تنزيل القائمة: $error';
  }

  @override
  String get bypassIntro =>
      'مواقع وعناوين تتجاوز VPN. إدخال في كل سطر: ‎example.com‎، ‎full:www.example.com‎، ‎keyword:bank‎، ‎regexp:…‎، ‎203.0.113.7‎ أو ‎10.0.0.0/8‎. تُتجاهل الأسطر التي تبدأ بـ #.';

  @override
  String get bypassNoLists => 'لا توجد قوائم بعد.';

  @override
  String get bypassTypeList => 'اكتب قائمة';

  @override
  String entriesCount(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count إدخال',
      many: '$count إدخالًا',
      few: '$count إدخالات',
      two: 'إدخالان',
      one: 'إدخال واحد',
      zero: 'لا إدخالات',
    );
    return '$_temp0';
  }

  @override
  String get bypassOnlyTitle => 'المدرجة فقط تستخدم VPN';

  @override
  String get bypassOnlyOn => 'البقية مباشرة.';

  @override
  String get bypassOnlyOff => 'المدرجة مباشرة والبقية عبر VPN.';

  @override
  String get bypassNothing => 'لا توجد قائمة مفعّلة.';

  @override
  String bypassInUse(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count إدخال قيد الاستخدام',
      many: '$count إدخالًا قيد الاستخدام',
      few: '$count إدخالات قيد الاستخدام',
      two: 'إدخالان قيد الاستخدام',
      one: 'إدخال واحد قيد الاستخدام',
      zero: 'لا إدخالات قيد الاستخدام',
    );
    return '$_temp0. يُطبَّق عند الاتصال التالي.';
  }

  @override
  String get bypassHttpLink => 'أدخل رابط http(s)';

  @override
  String bypassServerAnswered(int code) {
    return 'ردّ الخادم بالرمز $code';
  }

  @override
  String get bypassTooLarge => 'القائمة أكبر من 5 ميغابايت';

  @override
  String get bypassDownloadTitle => 'تنزيل قائمة';

  @override
  String get bypassNewList => 'قائمة جديدة';

  @override
  String get bypassEditList => 'تعديل القائمة';

  @override
  String get bypassName => 'الاسم';

  @override
  String get bypassEntriesLabel => 'الإدخالات، واحد في كل سطر';

  @override
  String get bypassNoUsable => 'لا توجد إدخالات صالحة بعد.';

  @override
  String bypassSummary(String entries, String names, String ranges) {
    return '$entries · $names، $ranges.';
  }

  @override
  String bypassNames(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count اسم',
      many: '$count اسمًا',
      few: '$count أسماء',
      two: 'اسمان',
      one: 'اسم واحد',
      zero: 'لا أسماء',
    );
    return '$_temp0';
  }

  @override
  String bypassRanges(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count عنوان أو نطاق',
      many: '$count عنوانًا أو نطاقًا',
      few: '$count عناوين أو نطاقات',
      two: 'عنوانان أو نطاقان',
      one: 'عنوان أو نطاق واحد',
      zero: 'لا عناوين',
    );
    return '$_temp0';
  }

  @override
  String bypassIgnored(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: 'سيُتجاهل $count سطر:',
      many: 'سيُتجاهل $count سطرًا:',
      few: 'ستُتجاهل $count أسطر:',
      two: 'سيُتجاهل سطران:',
      one: 'سيُتجاهل سطر واحد:',
    );
    return '$_temp0';
  }

  @override
  String bypassMoreIssues(int count) {
    return '… و$count أخرى';
  }

  @override
  String get bypassReconnect => 'أعد الاتصال لاستخدام القائمة الجديدة';

  @override
  String get bypassDefaultName => 'قائمتي';

  @override
  String get bypassPastedName => 'قائمة ملصوقة';

  @override
  String get settingsTitle => 'الإعدادات';

  @override
  String get settingsServers => 'الخوادم';

  @override
  String get manageServers => 'إدارة الخوادم';

  @override
  String get manageServersHint => 'الخوادم والاشتراكات';

  @override
  String get settingsConnection => 'الاتصال';

  @override
  String get vpnMode => 'وضع VPN (كل حركة البيانات)';

  @override
  String get vpnModeHint => 'متوقف: وكيل فقط، دون إذن VPN';

  @override
  String get systemProxy => 'ضبط وكيل النظام';

  @override
  String get systemProxyHint => 'ضبط وكيل النظام أثناء الاتصال';

  @override
  String get autoSwitch => 'التبديل التلقائي عند الفشل';

  @override
  String get autoSwitchHint => 'جرّب الخادم التالي عند تعطل الحالي';

  @override
  String get bypassLan => 'تجاوز الشبكة المحلية';

  @override
  String get bypassLanHint =>
      'الوصول إلى الأجهزة المحلية مباشرة (‎192.168.x.x‎، ‎10.x.x.x‎)';

  @override
  String get bypassListsHint => 'مواقع وعناوين تتجاوز VPN';

  @override
  String bypassListsInUse(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count إدخال قيد الاستخدام',
      many: '$count إدخالًا قيد الاستخدام',
      few: '$count إدخالات قيد الاستخدام',
      two: 'إدخالان قيد الاستخدام',
      one: 'إدخال واحد قيد الاستخدام',
      zero: 'لا إدخالات قيد الاستخدام',
    );
    return '$_temp0';
  }

  @override
  String get alwaysOnTitle => 'VPN دائم التشغيل وقاطع الاتصال';

  @override
  String get alwaysOnHint =>
      'إعادة الاتصال عند التشغيل وحظر الحركة إذا انقطع VPN. فعّله من إعدادات VPN في أندرويد (الترس بجوار Omnitun).';

  @override
  String get openVpnSettingsFallback =>
      'الإعدادات ← الشبكة ← VPN ← الترس بجوار Omnitun';

  @override
  String get splitTunneling => 'النفق المقسَّم';

  @override
  String splitSummaryOnly(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count تطبيق محدد فقط يستخدم VPN',
      many: '$count تطبيقًا محددًا فقط يستخدم VPN',
      few: '$count تطبيقات محددة فقط تستخدم VPN',
      two: 'تطبيقان محددان فقط يستخدمان VPN',
      one: 'تطبيق واحد محدد فقط يستخدم VPN',
    );
    return '$_temp0';
  }

  @override
  String splitSummaryBypass(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count تطبيق يتجاوز VPN',
      many: '$count تطبيقًا يتجاوز VPN',
      few: '$count تطبيقات تتجاوز VPN',
      two: 'تطبيقان يتجاوزان VPN',
      one: 'تطبيق واحد يتجاوز VPN',
    );
    return '$_temp0';
  }

  @override
  String get splitSummaryOff => 'جميع التطبيقات تستخدم VPN';

  @override
  String get remoteDns => 'DNS البعيد';

  @override
  String get remoteDnsHelp =>
      'DNS المستخدم عبر النفق، مثل ‎https://1.1.1.1/dns-query‎ أو ‎tls://1.1.1.1‎';

  @override
  String get settingsApp => 'التطبيق';

  @override
  String get appearance => 'المظهر';

  @override
  String get themeDark => 'داكن';

  @override
  String get themeLight => 'فاتح';

  @override
  String get themeSystem => 'حسب النظام';

  @override
  String get language => 'اللغة';

  @override
  String get languageSystem => 'افتراضي النظام';

  @override
  String get stopOnClose => 'إيقاف VPN عند إغلاق التطبيق';

  @override
  String aboutLine(String engine) {
    return 'رخصة MIT · المحرك: $engine';
  }

  @override
  String get splitTitle => 'النفق المقسَّم';

  @override
  String get splitAll => 'كل التطبيقات';

  @override
  String get splitOnlySelected => 'المحددة فقط';

  @override
  String get splitBypassSelected => 'تجاوز المحددة';

  @override
  String get splitHelpOffMode => 'كل التطبيقات تستخدم VPN.';

  @override
  String get splitHelpOnlyNone => 'لم تُحدَّد تطبيقات: الكل يستخدم VPN.';

  @override
  String splitHelpOnly(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count تطبيق فقط تستخدم VPN.',
      many: '$count تطبيقًا فقط تستخدم VPN.',
      few: '$count تطبيقات فقط تستخدم VPN.',
      two: 'تطبيقان فقط يستخدمان VPN.',
      one: 'تطبيق واحد فقط يستخدم VPN.',
    );
    return '$_temp0';
  }

  @override
  String splitHelpBypass(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count تطبيق يتصل مباشرة.',
      many: '$count تطبيقًا يتصل مباشرة.',
      few: '$count تطبيقات تتصل مباشرة.',
      two: 'تطبيقان يتصلان مباشرة.',
      one: 'تطبيق واحد يتصل مباشرة.',
    );
    return '$_temp0';
  }

  @override
  String get searchApps => 'بحث في التطبيقات';

  @override
  String get showSystemApps => 'إظهار تطبيقات النظام';

  @override
  String couldNotListApps(String error) {
    return 'تعذّر عرض التطبيقات: $error';
  }

  @override
  String get scanTitle => 'مسح رمز QR';

  @override
  String get scanTorch => 'المصباح';

  @override
  String scanCameraUnavailable(String code) {
    return 'الكاميرا غير متاحة: $code';
  }

  @override
  String get scanPoint => 'وجّه الكاميرا إلى رمز QR الخاص بالإعدادات';

  @override
  String get credTitle => 'تسجيل الدخول';

  @override
  String credNeeds(String name) {
    return 'يحتاج $name إلى اسم مستخدم وكلمة مرور.';
  }

  @override
  String get credUsername => 'اسم المستخدم';

  @override
  String get credPassword => 'كلمة المرور';

  @override
  String get logsEmpty => 'لا توجد سجلات بعد.';

  @override
  String get profilesTitle => 'إدارة الخوادم';

  @override
  String get addSubscriptionTooltip => 'إضافة اشتراك';

  @override
  String get testLatencyTooltip => 'اختبار زمن الاستجابة';

  @override
  String get profilesEmptyHint =>
      'لا توجد ملفات تعريف. اضغط «استيراد» لإضافة واحد.';

  @override
  String get profilesDragHint =>
      'اضغط مطولًا واسحب لإعادة الترتيب. التبديل التلقائي يتبع هذا الترتيب.';

  @override
  String get myProfiles => 'ملفاتي';

  @override
  String subRefreshed(String name, int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count ملف تعريف',
      many: '$count ملف تعريف',
      few: '$count ملفات تعريف',
      two: 'ملفا تعريف',
      one: 'ملف تعريف واحد',
      zero: 'لا ملفات تعريف',
    );
    return '$name: $_temp0';
  }

  @override
  String get refreshTooltip => 'تحديث';

  @override
  String get subDeleteWithProfiles => 'حذف الاشتراك وملفات تعريفه';

  @override
  String subUsageOf(String used, String total) {
    return '$used من $total';
  }

  @override
  String subUsed(String used) {
    return 'المستخدم $used';
  }

  @override
  String get subExpired => 'منتهي';

  @override
  String get subExpiresToday => 'ينتهي اليوم';

  @override
  String subExpiresIn(int days) {
    String _temp0 = intl.Intl.pluralLogic(
      days,
      locale: localeName,
      other: 'ينتهي خلال $days يوم',
      many: 'ينتهي خلال $days يومًا',
      few: 'ينتهي خلال $days أيام',
      two: 'ينتهي خلال يومين',
      one: 'ينتهي خلال يوم واحد',
    );
    return '$_temp0';
  }

  @override
  String subUpdated(String ago) {
    return 'حُدّث $ago';
  }

  @override
  String get agoJustNow => 'الآن';

  @override
  String agoMinutes(int count) {
    return 'منذ $count دقيقة';
  }

  @override
  String agoHours(int count) {
    return 'منذ $count ساعة';
  }

  @override
  String agoDays(int count) {
    return 'منذ $count يوم';
  }

  @override
  String subRefreshFailed(String error) {
    return 'فشل آخر تحديث: $error';
  }

  @override
  String get importTitle => 'استيراد الإعدادات';

  @override
  String get importDescription =>
      'الصق رابطًا أو إعدادًا أو اشتراكًا، أو افتح ملفًا أو امسح رمز QR.';

  @override
  String get importScanQr => 'مسح QR';

  @override
  String get importOpenFile => 'فتح ملف';

  @override
  String get importPaste => 'لصق';

  @override
  String get importSubscription => 'اشتراك';

  @override
  String get importTor => 'Tor';

  @override
  String get importAddTor => 'إضافة Tor';

  @override
  String get importTorDirect => 'Tor (مباشر)';

  @override
  String get importTorSnowflake => 'Tor + جسر Snowflake';

  @override
  String get importTorObfs4 => 'Tor + جسور obfs4';

  @override
  String importedProfiles(int added, int dupes) {
    String _temp0 = intl.Intl.pluralLogic(
      added,
      locale: localeName,
      other: 'تم استيراد $added ملف تعريف',
      many: 'تم استيراد $added ملف تعريف',
      few: 'تم استيراد $added ملفات تعريف',
      two: 'تم استيراد ملفي تعريف',
      one: 'تم استيراد ملف تعريف واحد',
      zero: 'لم يُستورد شيء',
    );
    String _temp1 = intl.Intl.pluralLogic(
      dupes,
      locale: localeName,
      other: '، منها $dupes موجود مسبقًا',
      zero: '',
    );
    return '$_temp0$_temp1';
  }

  @override
  String importNothing(String reason) {
    return 'لم يُستورد شيء: $reason';
  }

  @override
  String importPartial(int added, int failed) {
    return 'تم استيراد $added، وفشل $failed';
  }

  @override
  String get subAddTitle => 'إضافة اشتراك';

  @override
  String get subEditTitle => 'تعديل الاشتراك';

  @override
  String get subUrlLabel => 'رابط الاشتراك';

  @override
  String get subUrlHint =>
      'https://provider.example/sub/… أو nordvpn://<الرمز>';

  @override
  String get subNameLabel => 'الاسم (اختياري)';

  @override
  String get subNameHelp => 'الافتراضي هو عنوان المزوّد أو اسم المضيف';

  @override
  String get subAuthLabel => 'المصادقة';

  @override
  String get authNone => 'بلا';

  @override
  String get authBasic => 'اسم المستخدم وكلمة المرور (Basic)';

  @override
  String get authBearer => 'رمز Bearer';

  @override
  String get authHeader => 'ترويسة مخصصة';

  @override
  String get authQuery => 'رمز في الرابط';

  @override
  String get subUsername => 'اسم المستخدم';

  @override
  String get subPassword => 'كلمة المرور';

  @override
  String get subHeaderName => 'اسم الترويسة';

  @override
  String get subHeaderValue => 'قيمة الترويسة';

  @override
  String get subParamName => 'اسم المعامل';

  @override
  String get subToken => 'الرمز';

  @override
  String get subDownloadLabel => 'التنزيل';

  @override
  String get subDownloadHelp => 'المزوّد محجوب؟ يستطيع VPN جلبه';

  @override
  String get subViaDirect => 'مباشرة';

  @override
  String get subViaAuto => 'مباشرة، أو عبر VPN إن كان محجوبًا';

  @override
  String get subViaTunnel => 'دائمًا عبر VPN (اتصل أولًا)';

  @override
  String get subErrUrl =>
      'أدخل رابط اشتراك http(s) (أو nordvpn://<رمز الوصول>)';

  @override
  String get subErrHeader => 'أدخل اسم الترويسة';

  @override
  String get subErrToken => 'أدخل الرمز';

  @override
  String get subAddAndDownload => 'إضافة وتنزيل';

  @override
  String get subSaveAndRefresh => 'حفظ وتحديث';

  @override
  String subUpdatedToast(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count ملف تعريف',
      many: '$count ملف تعريف',
      few: '$count ملفات تعريف',
      two: 'ملفا تعريف',
      one: 'ملف تعريف واحد',
      zero: 'لا ملفات تعريف',
    );
    return 'تم تحديث الاشتراك: $_temp0';
  }

  @override
  String get flName => 'الاسم';

  @override
  String get flServer => 'الخادم';

  @override
  String get flPort => 'المنفذ';

  @override
  String get flPassword => 'كلمة المرور';

  @override
  String get flUsername => 'اسم المستخدم';

  @override
  String get flUuid => 'UUID';

  @override
  String get flSecurity => 'الأمان';

  @override
  String get flFlow => 'Flow';

  @override
  String get flCipher => 'الشيفرة';

  @override
  String get flEncryption => 'التشفير';

  @override
  String get flAlterId => 'Alter id';

  @override
  String get flTlsServerName => 'اسم خادم TLS ‏(SNI)';

  @override
  String get flTlsFingerprint => 'بصمة TLS';

  @override
  String get flAlpn => 'ALPN';

  @override
  String get flSkipCertCheck => 'تخطي فحص الشهادة';

  @override
  String get flPinnedCert => 'الشهادة المثبّتة (PEM)';

  @override
  String get flRealityPublicKey => 'المفتاح العام لـ Reality';

  @override
  String get flRealityShortId => 'المعرّف القصير لـ Reality';

  @override
  String get flTransport => 'النقل';

  @override
  String get flHostHeader => 'ترويسة Host';

  @override
  String get flPath => 'المسار';

  @override
  String get flGrpcService => 'اسم خدمة gRPC';

  @override
  String get flPlugin => 'الإضافة';

  @override
  String get flPluginOptions => 'خيارات الإضافة';

  @override
  String get flObfuscation => 'التمويه';

  @override
  String get flObfsPassword => 'كلمة مرور التمويه';

  @override
  String get flUploadMbps => 'الرفع (ميغابت/ث)';

  @override
  String get flDownloadMbps => 'التنزيل (ميغابت/ث)';

  @override
  String get flCongestion => 'التحكم بالازدحام';

  @override
  String get flUdpRelay => 'وضع ترحيل UDP';

  @override
  String get flUseQuic => 'استخدام HTTP/3 ‏(QUIC)';

  @override
  String get flTunnelAddress => 'عنوانك في النفق';

  @override
  String get flRoutedNetworks => 'الشبكات الموجَّهة';

  @override
  String get flDnsServers => 'خوادم DNS';

  @override
  String get flMtu => 'MTU';

  @override
  String get flKeepalive => 'Keepalive (ثوانٍ)';

  @override
  String get flPrivateKey => 'المفتاح الخاص';

  @override
  String get flServerPublicKey => 'المفتاح العام للخادم';

  @override
  String get flPresharedKey => 'المفتاح المشترك مسبقًا';

  @override
  String get flReservedBytes => 'البايتات المحجوزة';

  @override
  String get flAuthentication => 'المصادقة';

  @override
  String get flAuthDigest => 'ملخّص المصادقة';

  @override
  String get flCaCert => 'شهادة CA ‏(PEM)';

  @override
  String get flClientCert => 'شهادة العميل (PEM)';

  @override
  String get flClientKey => 'مفتاح العميل (PEM)';

  @override
  String get flControlWrap => 'حماية قناة التحكم';

  @override
  String get flControlWrapKey => 'مفتاح قناة التحكم';

  @override
  String get flKeyDirection => 'اتجاه المفتاح';

  @override
  String get flRemoteCertType => 'نوع شهادة الخادم';

  @override
  String get flVerifyServerName => 'التحقق من اسم الخادم';

  @override
  String get flYourIdentity => 'هويتك';

  @override
  String get flServerIdentity => 'هوية الخادم';

  @override
  String get flServerCertTrust => 'شهادة الخادم الموثوقة (PEM)';

  @override
  String get flUseSystemIke => 'استخدام عميل IKEv2 المدمج في الهاتف';

  @override
  String get flUseTcp => 'استخدام TCP';

  @override
  String get flHostKeys => 'مفاتيح المضيف';

  @override
  String get flPrivateKeyPem => 'المفتاح الخاص (PEM)';

  @override
  String get flKeyPassphrase => 'عبارة مرور المفتاح';

  @override
  String get flBridges => 'الجسور';

  @override
  String get flBridgeLines => 'أسطر الجسور';

  @override
  String get flJc => 'Jc (عدد الحزم العشوائية)';

  @override
  String get flSignaturePacket => 'I1 (حزمة التوقيع)';

  @override
  String get flControlledJunk => 'J1 (حزم عشوائية متحكَّم بها)';

  @override
  String get hpAuthorizedKeys => 'بصيغة authorized_keys؛ الفارغ يقبل أي مفتاح';

  @override
  String get hpIkeTcp => 'نقل IKE وESP عبر TCP للشبكات التي تحجب UDP';

  @override
  String get hpDefaultsToServer => 'الافتراضي هو عنوان الخادم';

  @override
  String get hpTorBridges =>
      'الفارغ يتصل بـ Tor مباشرة، أو يستخدم أسطر الجسور أدناه';

  @override
  String get hpExampleAddress => 'مثلًا 10.8.0.2/32';

  @override
  String get hpExamplePlugin => 'مثلًا shadow-tls أو v2ray-plugin';

  @override
  String get hpHostOrIp => 'اسم المضيف أو عنوان IP';

  @override
  String get hpNeededReality => 'مطلوب عندما يكون الأمان reality';

  @override
  String get hpOnlySelfSigned =>
      'فقط للخوادم ذات الشهادة الموقّعة ذاتيًا التي لا يمكنك تثبيتها';

  @override
  String get hpSafeSstp => 'آمن مع SSTP: تبادل كلمة المرور يثبت هوية الخادم';

  @override
  String get hpHysteriaObfs =>
      'salamander لـ Hysteria 2؛ وكلمة مرور التمويه لـ Hysteria 1';

  @override
  String get hpTrustCert => 'الوثوق بهذه الشهادة بدلًا من جهات النظام';

  @override
  String get hpWhichBrowser => 'مصافحة TLS لأي متصفح تُحاكى';

  @override
  String get hpWarpReserved => 'ثلاثة أرقام كما تُدرج في إعدادات WARP';

  @override
  String get hpOneBridgePerLine =>
      'جسر واحد في كل سطر كما في torrc لكن بدون «Bridge »';

  @override
  String get exitTitle => 'إظهار عنوان الخروج والبلد';

  @override
  String get exitHint =>
      'يسأل api.country.is عن عنوانك وبلدك، حتى عند عدم الاتصال.';

  @override
  String exitIpLabel(String ip) {
    return 'IP‎ $ip';
  }

  @override
  String connectedTo(String place) {
    return 'متصل بـ $place';
  }

  @override
  String get noticesTitle => 'تراخيص المصدر المفتوح';

  @override
  String get noticesIntro =>
      'يُوزَّع Omnitun بترخيص MIT. ويتضمن البرامج التالية، كلٌّ بترخيصه الخاص.';

  @override
  String noticesMore(int count) {
    return '+$count أخرى';
  }

  @override
  String noticesPackages(int count) {
    return '$count حزمة';
  }

  @override
  String get noticesLoadFailed => 'تعذّرت قراءة قائمة التراخيص.';

  @override
  String formRequired(String label) {
    return '$label مطلوب';
  }

  @override
  String formNotNumber(String label) {
    return 'يجب أن يكون $label عددًا صحيحًا';
  }

  @override
  String formNotBool(String label) {
    return 'يجب أن يكون $label مفعّلًا أو متوقفًا';
  }

  @override
  String formBadChoice(String label, String value, String choices) {
    return '$label: ‎$value‎ ليس من بين $choices';
  }

  @override
  String get formRealityOnlyVless => 'Reality‎ متاح فقط مع VLESS';

  @override
  String formUnknownSecurity(String value) {
    return 'حماية غير معروفة: ‎$value';
  }

  @override
  String get formReservedBytes =>
      'يجب أن تكون البايتات المحجوزة أعدادًا من 0 إلى 255';

  @override
  String get formRealityKeyRequired =>
      'المفتاح العام لـ Reality‎ مطلوب عندما تكون الحماية Reality';

  @override
  String get importUnsupported =>
      'هذا ليس رابط خادم أو اشتراكًا أو إعدادًا يستطيع Omnitun‎ قراءته';

  @override
  String importUnsupportedScheme(String scheme) {
    return 'الروابط التي تبدأ بـ ‎$scheme://‎ غير مدعومة';
  }

  @override
  String get importXrayServer =>
      'هذا إعداد خادم Xray‎. استورده على الخادم (‎omnitun-server import xray‎) لا كملف تعريف عميل';

  @override
  String get subAuthFailed =>
      'رفض الاشتراك تسجيل الدخول. تحقق من اسم المستخدم أو كلمة المرور أو الرمز';

  @override
  String subHttp(String status) {
    return 'ردّ خادم الاشتراك بـ ‎HTTP $status';
  }

  @override
  String get subTooLarge => 'حجم الاشتراك أكبر من 10 ميغابايت';

  @override
  String get subBadUrl => 'هذا ليس عنوان اشتراك صالحًا';

  @override
  String get bypassIssueEmptyExpression => 'تعبير فارغ';

  @override
  String get bypassIssueBadExpression => 'تعبير غير صالح';

  @override
  String get bypassIssueBadName => 'اسم غير صالح';

  @override
  String get bypassIssueGeo =>
      'القوائم الجغرافية غير مدعومة؛ اذكر النطاقات أو المدى';

  @override
  String get bypassIssueBadEntry => 'ليس نطاقًا ولا عنوانًا ولا مدى';

  @override
  String bypassIssueTooMany(int limit) {
    return 'أكثر من $limit إدخالًا: يُتجاهل الباقي';
  }

  @override
  String get torStarting => 'جارٍ تشغيل ‎Tor';

  @override
  String torBootstrap(int percent) {
    return 'جارٍ الاتصال بـ ‎Tor‎ $percent%';
  }

  @override
  String get torCircuit => 'جارٍ إنشاء مسار ‎Tor';

  @override
  String torRetry(int attempt, int attempts) {
    return 'إعادة محاولة ‎Tor‎ ($attempt/$attempts)';
  }

  @override
  String get importNoServer => 'لم يُعثر على أي خادم صالح في هذا النص';
}
