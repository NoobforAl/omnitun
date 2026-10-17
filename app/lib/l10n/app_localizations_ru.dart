// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Russian (`ru`).
class AppLocalizationsRu extends AppLocalizations {
  AppLocalizationsRu([String locale = 'ru']) : super(locale);

  @override
  String get appName => 'Omnitun';

  @override
  String get actionSave => 'Сохранить';

  @override
  String get actionCancel => 'Отмена';

  @override
  String get actionDelete => 'Удалить';

  @override
  String get actionEdit => 'Изменить';

  @override
  String get actionReset => 'Сбросить';

  @override
  String get actionSkip => 'Пропустить';

  @override
  String get actionNext => 'Далее';

  @override
  String get actionImport => 'Импорт';

  @override
  String get actionDownload => 'Скачать';

  @override
  String get actionCopyAll => 'Копировать всё';

  @override
  String get actionClear => 'Очистить';

  @override
  String get tooltipMore => 'Ещё';

  @override
  String get tooltipShow => 'Показать';

  @override
  String get tooltipHide => 'Скрыть';

  @override
  String get searchHint => 'Поиск';

  @override
  String get tooltipLogs => 'Журнал';

  @override
  String get tooltipSettings => 'Настройки';

  @override
  String get statusConnected => 'Подключено';

  @override
  String get statusConnecting => 'Подключение…';

  @override
  String get statusDisconnecting => 'Отключение…';

  @override
  String get statusError => 'Не удалось подключиться';

  @override
  String get statusNoInternet => 'Нет подключения к интернету';

  @override
  String get statusDisconnected => 'Не подключено';

  @override
  String get stagePermission => 'Ожидание разрешения VPN';

  @override
  String get stageNetworkCheck => 'Проверка подключения к интернету';

  @override
  String get stagePrepare => 'Подготовка конфигурации';

  @override
  String get stageEngineStart => 'Запуск ядра';

  @override
  String get stageTunnelUp => 'Открытие туннеля';

  @override
  String get stageHandshake => 'Связь с сервером';

  @override
  String get stageConnected => 'Подключено';

  @override
  String get buttonConnect => 'Подключить';

  @override
  String get buttonDisconnect => 'Отключить';

  @override
  String get buttonCancel => 'Отмена';

  @override
  String get locationLabel => 'Расположение';

  @override
  String get noServers => 'Нет серверов';

  @override
  String get noServersHint => 'Добавьте сервер, чтобы начать';

  @override
  String get automatic => 'Автоматически';

  @override
  String get automaticHint => 'Самый быстрый сервер, меняется при сбое';

  @override
  String get anyServerInCountry => 'Любой сервер в этой стране';

  @override
  String get torNetwork => 'Сеть Tor';

  @override
  String get emptyTitle => 'Профилей пока нет';

  @override
  String get emptyHint => 'Импортируйте конфигурацию, ссылку или подписку.';

  @override
  String get importConfig => 'Импорт конфигурации';

  @override
  String failoverToast(String to, String from) {
    return 'Переключено на $to — $from перестал отвечать';
  }

  @override
  String get unknownLocation => 'Неизвестное расположение';

  @override
  String errorFailed(String stage, String reason) {
    return '$stage: не удалось — $reason';
  }

  @override
  String get errTimeout => 'сервер не ответил вовремя';

  @override
  String get errRefused => 'сервер отклонил соединение';

  @override
  String get errReset => 'соединение было сброшено сетью';

  @override
  String get errNoRoute => 'сервер недоступен из этой сети';

  @override
  String get errNetworkUnreachable => 'сеть недоступна';

  @override
  String get errDns => 'не удалось определить адрес сервера';

  @override
  String get errTls => 'сертификат безопасности сервера отклонён';

  @override
  String get errAuth => 'имя пользователя или пароль отклонены';

  @override
  String get errClosed => 'сервер закрыл соединение';

  @override
  String get errCertDate =>
      'сертификат сервера просрочен или ещё не действует (проверьте также дату и время на телефоне)';

  @override
  String get errCertUntrusted =>
      'сертификат сервера подписан недоверенным центром';

  @override
  String get errCertName =>
      'сертификат сервера выдан для другого имени, чем то, к которому вы подключаетесь';

  @override
  String get errTlsKeys =>
      'общий ключ (tls-auth или tls-crypt) не совпадает с ключом сервера';

  @override
  String get errReality =>
      'сервер отклонил рукопожатие Reality (проверьте публичный ключ, short ID и имя сервера)';

  @override
  String get errNoHandshake =>
      'сервер не ответил на рукопожатие (проверьте ключи, адрес и открыт ли UDP-порт)';

  @override
  String get errUdpBlocked =>
      'сервер не ответил по UDP; сеть может блокировать его, попробуйте сервер на TCP';

  @override
  String get errPortInUse => 'локальный порт занят другим приложением';

  @override
  String get errTorTimeout =>
      'Tor не успел подключиться. Возможно, эта сеть блокирует Tor; используйте мост (Snowflake или obfs4)';

  @override
  String get errTorBridgeTimeout =>
      'Tor не успел подключиться. Мост может быть заблокирован или перегружен; попробуйте другой тип моста (Snowflake или obfs4)';

  @override
  String get errTorBadBridge =>
      'строка моста Tor недействительна (нужны транспорт, адрес и отпечаток)';

  @override
  String get errTorMissing => 'в этой сборке нет клиента Tor';

  @override
  String get errPermission => 'разрешение VPN не выдано';

  @override
  String get errCoreLost => 'потеряно соединение со службой ядра';

  @override
  String get selectLocation => 'Выбор расположения';

  @override
  String get testSpeedTooltip => 'Проверить скорость всех серверов';

  @override
  String get addServersTooltip => 'Добавить серверы';

  @override
  String get menuImport => 'Импорт конфигурации, ссылки или QR';

  @override
  String get menuNewServer => 'Добавить сервер вручную';

  @override
  String get menuSubscription => 'Добавить подписку';

  @override
  String get menuManage => 'Серверы и подписки';

  @override
  String get menuSetDefault => 'Сделать основным';

  @override
  String get badgeDefault => 'Основной';

  @override
  String get menuDeleteAll => 'Удалить все серверы';

  @override
  String get deleteAllTitle => 'Удалить все серверы?';

  @override
  String get deleteAllBody =>
      'Все добавленные вами серверы и подписки будут удалены. Встроенное подключение Tor останется, и приложение вернётся к нему.';

  @override
  String get menuRestoreTor => 'Восстановить Tor по умолчанию';

  @override
  String get deletedAllToast => 'Все серверы удалены';

  @override
  String defaultSetToast(String name) {
    return '$name теперь основной';
  }

  @override
  String serversCount(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count сервера',
      many: '$count серверов',
      few: '$count сервера',
      one: '$count сервер',
    );
    return '$_temp0';
  }

  @override
  String get otherGroup => 'Другие';

  @override
  String get torGroup => 'Tor';

  @override
  String get noServersYet => 'Серверов пока нет';

  @override
  String get importHintShort =>
      'Импортируйте файл конфигурации, ссылку или подписку.';

  @override
  String get showServers => 'Показать серверы';

  @override
  String get hideServers => 'Скрыть серверы';

  @override
  String get menuEditServer => 'Изменить';

  @override
  String get menuBridgeSettings => 'Настройки мостов';

  @override
  String get menuShare => 'Поделиться ссылкой или QR';

  @override
  String get menuSetCountry => 'Указать страну';

  @override
  String get menuCredentials => 'Имя пользователя и пароль';

  @override
  String needsSignIn(String address) {
    return '$address · нужен вход';
  }

  @override
  String get countryOfServer => 'Страна этого сервера';

  @override
  String get guessFromName => 'Определить по названию';

  @override
  String get latencyNotTested => 'Не проверялся';

  @override
  String get latencyUnreachable => 'Недоступен';

  @override
  String get latencyTimeout => 'таймаут';

  @override
  String latencyMs(int ms) {
    return '$ms мс';
  }

  @override
  String get unknownCountry => 'Неизвестная страна';

  @override
  String get addServerTitle => 'Добавить сервер';

  @override
  String get addServerIntro =>
      'Выберите протокол. Ссылка или QR быстрее: используйте «Импорт».';

  @override
  String newServerTitle(String label) {
    return 'Новый сервер $label';
  }

  @override
  String get editServerTitle => 'Изменение сервера';

  @override
  String get torBridgesTitle => 'Мосты Tor';

  @override
  String get subscriptionEditNote =>
      'Этот сервер из подписки: ваши изменения будут заменены при её обновлении.';

  @override
  String get moreOptions => 'Дополнительно';

  @override
  String get countryRow => 'Страна';

  @override
  String get countryAuto => 'Определяется по названию и адресу';

  @override
  String get alreadyInList => 'Этот сервер уже есть в списке';

  @override
  String get choiceDefault => 'По умолчанию';

  @override
  String get torBridgesNone => 'Нет (напрямую или строки ниже)';

  @override
  String get torBridgesSnowflake => 'Snowflake';

  @override
  String get torBridgesObfs4 => 'obfs4 (встроенные мосты)';

  @override
  String get listHelper => 'Разделяйте значения запятыми';

  @override
  String get fillPaste => 'Вставить';

  @override
  String get fillFile => 'Файл';

  @override
  String get fillScanQr => 'Сканировать QR';

  @override
  String get openTextFile => 'Открыть текстовый файл';

  @override
  String couldNotReadFile(String error) {
    return 'Не удалось прочитать файл: $error';
  }

  @override
  String fieldRequired(String label) {
    return '$label *';
  }

  @override
  String formNoEditor(String protocol) {
    return 'Нет редактора для $protocol';
  }

  @override
  String get shareScanHint => 'Отсканируйте в Omnitun или скопируйте ссылку.';

  @override
  String get shareCopy => 'Копировать ссылку';

  @override
  String get shareCopied => 'Ссылка скопирована';

  @override
  String get shareWarning =>
      'Содержит пароль или ключ сервера. Делитесь осторожно.';

  @override
  String shareFailed(String reason) {
    return 'Этим сервером нельзя поделиться ссылкой: $reason';
  }

  @override
  String get shareTooLong => 'Слишком длинно для QR-кода: скопируйте ссылку';

  @override
  String get obTitle1 => 'Приватно и просто';

  @override
  String get obText1 =>
      'Один спокойный экран для VPN, прокси и Tor. Выберите место и нажмите «Подключить».';

  @override
  String get obTitle2 => 'Добавьте серверы';

  @override
  String get obText2 =>
      'Импортируйте ссылку, QR-код, файл конфигурации или подписку. Или добавьте сервер вручную.';

  @override
  String get obTitle3Android => 'Android спросит один раз';

  @override
  String get obText3Android =>
      'Android попросит разрешить VPN-соединение. Разрешите, чтобы быть защищённым.';

  @override
  String get obTitle3Desktop => 'Всё готово';

  @override
  String get obText3Desktop =>
      'Запускается в режиме прокси без особых прав. Включите режим VPN в настройках, чтобы перехватывать весь трафик.';

  @override
  String get obAddFirst => 'Добавить первый сервер';

  @override
  String get bypassTitle => 'Списки исключений';

  @override
  String get bypassAddTooltip => 'Добавить список';

  @override
  String get bypassFromClipboard => 'Вставить из буфера обмена';

  @override
  String get bypassFromFile => 'Открыть файл .txt';

  @override
  String get bypassFromUrl => 'Скачать по ссылке';

  @override
  String get bypassType => 'Ввести вручную';

  @override
  String get bypassClipboardEmpty => 'Буфер обмена пуст';

  @override
  String get bypassPickFile => 'Открыть список исключений (.txt)';

  @override
  String bypassDownloadFailed(String error) {
    return 'Не удалось скачать список: $error';
  }

  @override
  String get bypassIntro =>
      'Сайты и адреса мимо VPN. По одному в строке: example.com, full:www.example.com, keyword:bank, regexp:…, 203.0.113.7 или 10.0.0.0/8. Строки с # игнорируются.';

  @override
  String get bypassNoLists => 'Списков пока нет.';

  @override
  String get bypassTypeList => 'Ввести список';

  @override
  String entriesCount(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count записи',
      many: '$count записей',
      few: '$count записи',
      one: '$count запись',
    );
    return '$_temp0';
  }

  @override
  String get bypassOnlyTitle => 'Через VPN только из списков';

  @override
  String get bypassOnlyOn => 'Остальное идёт напрямую.';

  @override
  String get bypassOnlyOff => 'Из списков — напрямую, остальное — через VPN.';

  @override
  String get bypassNothing => 'Ни один список не включён.';

  @override
  String bypassInUse(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count записи используются',
      many: '$count записей используются',
      few: '$count записи используются',
      one: '$count запись используется',
    );
    return '$_temp0. Применится при следующем подключении.';
  }

  @override
  String get bypassHttpLink => 'Введите ссылку http(s)';

  @override
  String bypassServerAnswered(int code) {
    return 'сервер ответил $code';
  }

  @override
  String get bypassTooLarge => 'список больше 5 МБ';

  @override
  String get bypassDownloadTitle => 'Скачать список';

  @override
  String get bypassNewList => 'Новый список';

  @override
  String get bypassEditList => 'Изменение списка';

  @override
  String get bypassName => 'Название';

  @override
  String get bypassEntriesLabel => 'Записи, по одной в строке';

  @override
  String get bypassNoUsable => 'Пригодных записей пока нет.';

  @override
  String bypassSummary(String entries, String names, String ranges) {
    return '$entries · $names, $ranges.';
  }

  @override
  String bypassNames(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count имени',
      many: '$count имён',
      few: '$count имени',
      one: '$count имя',
    );
    return '$_temp0';
  }

  @override
  String bypassRanges(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count адреса или диапазона',
      many: '$count адресов или диапазонов',
      few: '$count адреса или диапазона',
      one: '$count адрес или диапазон',
    );
    return '$_temp0';
  }

  @override
  String bypassIgnored(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count строки будут проигнорированы:',
      many: '$count строк будут проигнорированы:',
      few: '$count строки будут проигнорированы:',
      one: '$count строка будет проигнорирована:',
    );
    return '$_temp0';
  }

  @override
  String bypassMoreIssues(int count) {
    return '… и ещё $count';
  }

  @override
  String get bypassReconnect =>
      'Переподключитесь, чтобы применить новый список';

  @override
  String get bypassDefaultName => 'Мой список';

  @override
  String get bypassPastedName => 'Вставленный список';

  @override
  String get settingsTitle => 'Настройки';

  @override
  String get settingsServers => 'Серверы';

  @override
  String get manageServers => 'Серверы и подписки';

  @override
  String get manageServersHint => 'Серверы и подписки';

  @override
  String get settingsConnection => 'Подключение';

  @override
  String get vpnMode => 'Режим VPN (весь трафик)';

  @override
  String get vpnModeHint => 'Выкл.: только прокси, без разрешения VPN';

  @override
  String get systemProxy => 'Задать системный прокси';

  @override
  String get systemProxyHint =>
      'Включить системный прокси на время подключения';

  @override
  String get autoSwitch => 'Автопереключение при сбое';

  @override
  String get autoSwitchHint => 'При сбое пробовать следующий сервер';

  @override
  String get bypassLan => 'Локальная сеть мимо VPN';

  @override
  String get bypassLanHint =>
      'Локальные устройства напрямую (192.168.x.x, 10.x.x.x)';

  @override
  String get bypassListsHint => 'Сайты и адреса, которые идут мимо VPN';

  @override
  String bypassListsInUse(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: 'Используются $count записи',
      many: 'Используются $count записей',
      few: 'Используются $count записи',
      one: 'Используется $count запись',
    );
    return '$_temp0';
  }

  @override
  String get alwaysOnTitle => 'Постоянный VPN и блокировка без VPN';

  @override
  String get alwaysOnHint =>
      'Подключаться при загрузке и блокировать трафик, если VPN упал. Включите в настройках VPN Android (шестерёнка у Omnitun).';

  @override
  String get openVpnSettingsFallback =>
      'Настройки → Сеть → VPN → шестерёнка у Omnitun';

  @override
  String get splitTunneling => 'Раздельное туннелирование';

  @override
  String splitSummaryOnly(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: 'Через VPN работают только $count выбранных приложения',
      many: 'Через VPN работают только $count выбранных приложений',
      few: 'Через VPN работают только $count выбранных приложения',
      one: 'Через VPN работает только $count выбранное приложение',
    );
    return '$_temp0';
  }

  @override
  String splitSummaryBypass(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count приложения работают без VPN',
      many: '$count приложений работают без VPN',
      few: '$count приложения работают без VPN',
      one: '$count приложение работает без VPN',
    );
    return '$_temp0';
  }

  @override
  String get splitSummaryOff => 'Все приложения используют VPN';

  @override
  String get remoteDns => 'Удалённый DNS';

  @override
  String get remoteDnsHelp =>
      'DNS через туннель, например https://1.1.1.1/dns-query или tls://1.1.1.1';

  @override
  String get settingsApp => 'Приложение';

  @override
  String get appearance => 'Оформление';

  @override
  String get themeDark => 'Тёмная';

  @override
  String get themeLight => 'Светлая';

  @override
  String get themeSystem => 'Как в системе';

  @override
  String get language => 'Язык';

  @override
  String get languageSystem => 'Как в системе';

  @override
  String get stopOnClose => 'Отключать VPN при закрытии приложения';

  @override
  String aboutLine(String engine) {
    return 'Лицензия MIT · ядро: $engine';
  }

  @override
  String get splitTitle => 'Раздельное туннелирование';

  @override
  String get splitAll => 'Все приложения';

  @override
  String get splitOnlySelected => 'Только выбранные';

  @override
  String get splitBypassSelected => 'Выбранные мимо VPN';

  @override
  String get splitHelpOffMode => 'Все приложения используют VPN.';

  @override
  String get splitHelpOnlyNone => 'Приложения не выбраны: все используют VPN.';

  @override
  String splitHelpOnly(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: 'Через VPN только $count приложения.',
      many: 'Через VPN только $count приложений.',
      few: 'Через VPN только $count приложения.',
      one: 'Через VPN только $count приложение.',
    );
    return '$_temp0';
  }

  @override
  String splitHelpBypass(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count приложения идут напрямую.',
      many: '$count приложений идут напрямую.',
      few: '$count приложения идут напрямую.',
      one: '$count приложение идёт напрямую.',
    );
    return '$_temp0';
  }

  @override
  String get searchApps => 'Поиск приложений';

  @override
  String get showSystemApps => 'Показывать системные приложения';

  @override
  String couldNotListApps(String error) {
    return 'Не удалось получить список приложений: $error';
  }

  @override
  String get scanTitle => 'Сканирование QR-кода';

  @override
  String get scanTorch => 'Фонарик';

  @override
  String scanCameraUnavailable(String code) {
    return 'Камера недоступна: $code';
  }

  @override
  String get scanPoint => 'Наведите камеру на QR-код конфигурации';

  @override
  String get credTitle => 'Вход';

  @override
  String credNeeds(String name) {
    return 'Для $name нужны имя пользователя и пароль.';
  }

  @override
  String get credUsername => 'Имя пользователя';

  @override
  String get credPassword => 'Пароль';

  @override
  String get logsEmpty => 'Записей в журнале пока нет.';

  @override
  String get profilesTitle => 'Серверы и подписки';

  @override
  String get addSubscriptionTooltip => 'Добавить подписку';

  @override
  String get testLatencyTooltip => 'Проверить задержку';

  @override
  String get profilesEmptyHint =>
      'Профилей нет. Нажмите «Импорт», чтобы добавить.';

  @override
  String get profilesDragHint =>
      'Удерживайте и перетащите, чтобы изменить порядок. Переключение идёт в этом порядке.';

  @override
  String get myProfiles => 'Мои профили';

  @override
  String subRefreshed(String name, int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count профиля',
      many: '$count профилей',
      few: '$count профиля',
      one: '$count профиль',
    );
    return '$name: $_temp0';
  }

  @override
  String get refreshTooltip => 'Обновить';

  @override
  String get subDeleteWithProfiles => 'Удалить подписку и её профили';

  @override
  String subUsageOf(String used, String total) {
    return '$used из $total';
  }

  @override
  String subUsed(String used) {
    return 'использовано $used';
  }

  @override
  String get subExpired => 'истекла';

  @override
  String get subExpiresToday => 'истекает сегодня';

  @override
  String subExpiresIn(int days) {
    String _temp0 = intl.Intl.pluralLogic(
      days,
      locale: localeName,
      other: 'истекает через $days дня',
      many: 'истекает через $days дней',
      few: 'истекает через $days дня',
      one: 'истекает через $days день',
    );
    return '$_temp0';
  }

  @override
  String subUpdated(String ago) {
    return 'обновлено $ago';
  }

  @override
  String get agoJustNow => 'только что';

  @override
  String agoMinutes(int count) {
    return '$count мин назад';
  }

  @override
  String agoHours(int count) {
    return '$count ч назад';
  }

  @override
  String agoDays(int count) {
    return '$count дн назад';
  }

  @override
  String subRefreshFailed(String error) {
    return 'Последнее обновление не удалось: $error';
  }

  @override
  String get importTitle => 'Импорт конфигурации';

  @override
  String get importDescription =>
      'Вставьте ссылку, конфигурацию или подписку — или откройте файл или отсканируйте QR-код.';

  @override
  String get importScanQr => 'Сканировать QR';

  @override
  String get importOpenFile => 'Открыть файл';

  @override
  String get importPaste => 'Вставить';

  @override
  String get importSubscription => 'Подписка';

  @override
  String get importTor => 'Tor';

  @override
  String get importAddTor => 'Добавить Tor';

  @override
  String get importTorDirect => 'Tor (напрямую)';

  @override
  String get importTorSnowflake => 'Tor + мост Snowflake';

  @override
  String get importTorObfs4 => 'Tor + мосты obfs4';

  @override
  String importedProfiles(int added, int dupes) {
    String _temp0 = intl.Intl.pluralLogic(
      added,
      locale: localeName,
      other: 'Импортировано $added профиля',
      many: 'Импортировано $added профилей',
      few: 'Импортировано $added профиля',
      one: 'Импортирован $added профиль',
    );
    String _temp1 = intl.Intl.pluralLogic(
      dupes,
      locale: localeName,
      other: ', уже были: $dupes',
      zero: '',
    );
    return '$_temp0$_temp1';
  }

  @override
  String importNothing(String reason) {
    return 'Ничего не импортировано: $reason';
  }

  @override
  String importPartial(int added, int failed) {
    return 'Импортировано $added, не удалось $failed';
  }

  @override
  String get subAddTitle => 'Добавить подписку';

  @override
  String get subEditTitle => 'Изменение подписки';

  @override
  String get subUrlLabel => 'Ссылка подписки';

  @override
  String get subUrlHint =>
      'https://provider.example/sub/… или nordvpn://<токен>';

  @override
  String get subNameLabel => 'Название (необязательно)';

  @override
  String get subNameHelp => 'По умолчанию — название провайдера или имя хоста';

  @override
  String get subAuthLabel => 'Аутентификация';

  @override
  String get authNone => 'Нет';

  @override
  String get authBasic => 'Имя и пароль (Basic)';

  @override
  String get authBearer => 'Токен Bearer';

  @override
  String get authHeader => 'Свой заголовок';

  @override
  String get authQuery => 'Токен в ссылке';

  @override
  String get subUsername => 'Имя пользователя';

  @override
  String get subPassword => 'Пароль';

  @override
  String get subHeaderName => 'Имя заголовка';

  @override
  String get subHeaderValue => 'Значение заголовка';

  @override
  String get subParamName => 'Имя параметра';

  @override
  String get subToken => 'Токен';

  @override
  String get subDownloadLabel => 'Загрузка';

  @override
  String get subDownloadHelp =>
      'Провайдер заблокирован? VPN всё равно загрузит';

  @override
  String get subViaDirect => 'Напрямую';

  @override
  String get subViaAuto => 'Напрямую или через VPN, если заблокировано';

  @override
  String get subViaTunnel => 'Всегда через VPN (сначала подключитесь)';

  @override
  String get subErrUrl =>
      'Введите ссылку подписки http(s) (или nordvpn://<токен доступа>)';

  @override
  String get subErrHeader => 'Введите имя заголовка';

  @override
  String get subErrToken => 'Введите токен';

  @override
  String get subAddAndDownload => 'Добавить и скачать';

  @override
  String get subSaveAndRefresh => 'Сохранить и обновить';

  @override
  String subUpdatedToast(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count профиля',
      many: '$count профилей',
      few: '$count профиля',
      one: '$count профиль',
    );
    return 'Подписка обновлена: $_temp0';
  }

  @override
  String get flName => 'Название';

  @override
  String get flServer => 'Сервер';

  @override
  String get flPort => 'Порт';

  @override
  String get flPassword => 'Пароль';

  @override
  String get flUsername => 'Имя пользователя';

  @override
  String get flUuid => 'UUID';

  @override
  String get flSecurity => 'Защита';

  @override
  String get flFlow => 'Flow';

  @override
  String get flCipher => 'Шифр';

  @override
  String get flEncryption => 'Шифрование';

  @override
  String get flAlterId => 'Alter id';

  @override
  String get flTlsServerName => 'Имя сервера TLS (SNI)';

  @override
  String get flTlsFingerprint => 'Отпечаток TLS';

  @override
  String get flAlpn => 'ALPN';

  @override
  String get flSkipCertCheck => 'Не проверять сертификат';

  @override
  String get flPinnedCert => 'Закреплённый сертификат (PEM)';

  @override
  String get flRealityPublicKey => 'Открытый ключ Reality';

  @override
  String get flRealityShortId => 'Короткий id Reality';

  @override
  String get flTransport => 'Транспорт';

  @override
  String get flHostHeader => 'Заголовок Host';

  @override
  String get flPath => 'Путь';

  @override
  String get flGrpcService => 'Имя службы gRPC';

  @override
  String get flPlugin => 'Плагин';

  @override
  String get flPluginOptions => 'Параметры плагина';

  @override
  String get flObfuscation => 'Обфускация';

  @override
  String get flObfsPassword => 'Пароль обфускации';

  @override
  String get flUploadMbps => 'Отдача (Мбит/с)';

  @override
  String get flDownloadMbps => 'Загрузка (Мбит/с)';

  @override
  String get flCongestion => 'Управление перегрузкой';

  @override
  String get flUdpRelay => 'Режим ретрансляции UDP';

  @override
  String get flUseQuic => 'Использовать HTTP/3 (QUIC)';

  @override
  String get flTunnelAddress => 'Ваш адрес в туннеле';

  @override
  String get flRoutedNetworks => 'Маршрутизируемые сети';

  @override
  String get flDnsServers => 'Серверы DNS';

  @override
  String get flMtu => 'MTU';

  @override
  String get flKeepalive => 'Keepalive (секунды)';

  @override
  String get flPrivateKey => 'Закрытый ключ';

  @override
  String get flServerPublicKey => 'Открытый ключ сервера';

  @override
  String get flPresharedKey => 'Предварительный общий ключ';

  @override
  String get flReservedBytes => 'Зарезервированные байты';

  @override
  String get flAuthentication => 'Аутентификация';

  @override
  String get flAuthDigest => 'Дайджест аутентификации';

  @override
  String get flCaCert => 'Сертификат CA (PEM)';

  @override
  String get flClientCert => 'Сертификат клиента (PEM)';

  @override
  String get flClientKey => 'Ключ клиента (PEM)';

  @override
  String get flControlWrap => 'Защита канала управления';

  @override
  String get flControlWrapKey => 'Ключ канала управления';

  @override
  String get flKeyDirection => 'Направление ключа';

  @override
  String get flRemoteCertType => 'Тип сертификата сервера';

  @override
  String get flVerifyServerName => 'Проверять имя сервера';

  @override
  String get flYourIdentity => 'Ваш идентификатор';

  @override
  String get flServerIdentity => 'Идентификатор сервера';

  @override
  String get flServerCertTrust => 'Сертификат сервера, которому доверять (PEM)';

  @override
  String get flUseSystemIke => 'Использовать встроенный клиент IKEv2 телефона';

  @override
  String get flUseTcp => 'Использовать TCP';

  @override
  String get flHostKeys => 'Ключи хоста';

  @override
  String get flPrivateKeyPem => 'Закрытый ключ (PEM)';

  @override
  String get flKeyPassphrase => 'Парольная фраза ключа';

  @override
  String get flBridges => 'Мосты';

  @override
  String get flBridgeLines => 'Строки мостов';

  @override
  String get flJc => 'Jc (число мусорных пакетов)';

  @override
  String get flSignaturePacket => 'I1 (пакет-подпись)';

  @override
  String get flControlledJunk => 'J1 (управляемый мусор)';

  @override
  String get hpAuthorizedKeys =>
      'Формат authorized_keys; пусто — принимается любой ключ';

  @override
  String get hpIkeTcp =>
      'Передавать IKE и ESP по TCP для сетей, блокирующих UDP';

  @override
  String get hpDefaultsToServer => 'По умолчанию — адрес сервера';

  @override
  String get hpTorBridges =>
      'Пусто — прямое подключение к Tor или мосты из строк ниже';

  @override
  String get hpExampleAddress => 'Например, 10.8.0.2/32';

  @override
  String get hpExamplePlugin => 'Например, shadow-tls или v2ray-plugin';

  @override
  String get hpHostOrIp => 'Имя хоста или IP-адрес';

  @override
  String get hpNeededReality => 'Нужно, когда защита — reality';

  @override
  String get hpOnlySelfSigned =>
      'Только для серверов с самоподписанным сертификатом, который нельзя закрепить';

  @override
  String get hpSafeSstp =>
      'Для SSTP безопасно: обмен паролями подтверждает сервер';

  @override
  String get hpHysteriaObfs =>
      'salamander для Hysteria 2; пароль обфускации для Hysteria 1';

  @override
  String get hpTrustCert =>
      'Доверять этому сертификату вместо системных центров';

  @override
  String get hpWhichBrowser => 'Рукопожатие TLS какого браузера имитировать';

  @override
  String get hpWarpReserved => 'Три числа, как в конфигурациях WARP';

  @override
  String get hpOneBridgePerLine =>
      'Один мост на строку, как в torrc, но без «Bridge »';

  @override
  String get exitTitle => 'Показывать адрес и страну выхода';

  @override
  String get exitHint =>
      'Спрашивать у api.country.is ваш адрес и страну, в том числе без подключения.';

  @override
  String exitIpLabel(String ip) {
    return 'IP $ip';
  }

  @override
  String connectedTo(String place) {
    return 'Подключено: $place';
  }

  @override
  String get noticesTitle => 'Лицензии с открытым кодом';

  @override
  String get noticesIntro =>
      'Omnitun распространяется по лицензии MIT. В него входит ПО ниже, каждое под своей лицензией.';

  @override
  String noticesMore(int count) {
    return 'ещё $count';
  }

  @override
  String noticesPackages(int count) {
    return 'Пакетов: $count';
  }

  @override
  String get noticesLoadFailed => 'Не удалось прочитать список лицензий.';

  @override
  String formRequired(String label) {
    return 'Поле «$label» обязательно';
  }

  @override
  String formNotNumber(String label) {
    return 'Поле «$label» должно быть целым числом';
  }

  @override
  String formNotBool(String label) {
    return 'Поле «$label» должно быть включено или выключено';
  }

  @override
  String formBadChoice(String label, String value, String choices) {
    return '$label: «$value» нет среди вариантов: $choices';
  }

  @override
  String get formRealityOnlyVless => 'Reality доступен только для VLESS';

  @override
  String formUnknownSecurity(String value) {
    return 'Неизвестная защита: $value';
  }

  @override
  String get formReservedBytes =>
      'Зарезервированные байты должны быть числами от 0 до 255';

  @override
  String get formRealityKeyRequired =>
      'При защите Reality нужен открытый ключ Reality';

  @override
  String get importUnsupported =>
      'Это не ссылка на сервер, не подписка и не конфигурация, которую Omnitun может прочитать';

  @override
  String importUnsupportedScheme(String scheme) {
    return 'Ссылки, начинающиеся с $scheme://, не поддерживаются';
  }

  @override
  String get importXrayServer =>
      'Это конфигурация сервера Xray. Импортируйте её на сервере (omnitun-server import xray), а не как клиентский профиль';

  @override
  String get subAuthFailed =>
      'Подписка отклонила вход. Проверьте имя пользователя, пароль или токен';

  @override
  String subHttp(String status) {
    return 'Сервер подписки ответил HTTP $status';
  }

  @override
  String get subTooLarge => 'Подписка больше 10 МБ';

  @override
  String get subBadUrl => 'Это не похоже на адрес подписки';

  @override
  String get bypassIssueEmptyExpression => 'пустое выражение';

  @override
  String get bypassIssueBadExpression => 'недопустимое выражение';

  @override
  String get bypassIssueBadName => 'недопустимое имя';

  @override
  String get bypassIssueGeo =>
      'гео-списки не поддерживаются; перечислите домены или диапазоны';

  @override
  String get bypassIssueBadEntry => 'это не домен, не адрес и не диапазон';

  @override
  String bypassIssueTooMany(int limit) {
    return 'больше $limit записей: остальные игнорируются';
  }

  @override
  String get torStarting => 'Запуск Tor';

  @override
  String torBootstrap(int percent) {
    return 'Подключение к Tor: $percent%';
  }

  @override
  String get torCircuit => 'Построение цепочки Tor';

  @override
  String torRetry(int attempt, int attempts) {
    return 'Повторная попытка Tor ($attempt/$attempts)';
  }

  @override
  String get importNoServer =>
      'В этом тексте нет ни одного подходящего сервера';

  @override
  String get ipdbTitle => 'Офлайн-база стран';

  @override
  String get ipdbIntro =>
      'Скачайте базу один раз, и Omnitun сам определит страну сервера или адреса, не обращаясь к сайтам.';

  @override
  String ipdbSizeNote(String size) {
    return 'Около $size · свободные данные (общественное достояние)';
  }

  @override
  String get ipdbDownload => 'Скачать';

  @override
  String get ipdbContinue => 'Продолжить';

  @override
  String get ipdbPause => 'Пауза';

  @override
  String get ipdbUpdate => 'Обновить';

  @override
  String get ipdbDelete => 'Удалить';

  @override
  String get ipdbDeleteTitle => 'Удалить базу?';

  @override
  String get ipdbUse => 'Использовать базу';

  @override
  String get ipdbStepV4 => 'IPv4';

  @override
  String get ipdbStepV6 => 'IPv6';

  @override
  String get ipdbStepPrepare => 'Подготовка';

  @override
  String get ipdbPreparing => 'Подготовка…';

  @override
  String get ipdbReady => 'Готово';

  @override
  String ipdbStoppedAt(int percent) {
    return 'Остановлено на $percent%. «Продолжить» начнёт с того же места.';
  }

  @override
  String ipdbFailed(String reason) {
    return 'Не удалось скачать: $reason';
  }

  @override
  String ipdbReadyLine(String ranges, String size) {
    return '$ranges диапазонов · $size';
  }

  @override
  String ipdbVersion(String version, String date) {
    return 'Версия $version · $date';
  }

  @override
  String ipdbLeft(String time) {
    return 'Осталось $time';
  }

  @override
  String get ipdbSource =>
      'Данные: региональные интернет-реестры и анонсы ASN (CC0), проект ip-location-db.';

  @override
  String get ipdbSubNone => 'Определять страны, не обращаясь к сайтам';

  @override
  String ipdbSubReady(String size) {
    return 'Вкл. · $size';
  }

  @override
  String ipdbSubOff(String size) {
    return 'Выкл. · $size';
  }

  @override
  String ipdbSubDownloading(int percent) {
    return 'Загрузка $percent%';
  }

  @override
  String ipdbSubStopped(int percent) {
    return 'Остановлено на $percent%';
  }

  @override
  String get ipdbErrNetwork => 'нет соединения с серверами данных';

  @override
  String get ipdbErrMirrors => 'серверы данных прислали не тот файл';
}
