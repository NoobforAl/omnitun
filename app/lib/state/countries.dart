/// Countries (ISO 3166-1 alpha-2) and a best-effort guess of the country of a
/// server from its name and address. The guess is only a label for the
/// location picker; it never changes how a connection is made.
library;

import '../engine/engine.dart';

const _table = '''
AD:Andorra|AE:United Arab Emirates|AF:Afghanistan|AG:Antigua and Barbuda|AL:Albania|AM:Armenia|AO:Angola|AR:Argentina|AT:Austria|AU:Australia|AZ:Azerbaijan|
BA:Bosnia and Herzegovina|BB:Barbados|BD:Bangladesh|BE:Belgium|BF:Burkina Faso|BG:Bulgaria|BH:Bahrain|BI:Burundi|BJ:Benin|BN:Brunei|BO:Bolivia|BR:Brazil|BS:Bahamas|BT:Bhutan|BW:Botswana|BY:Belarus|BZ:Belize|
CA:Canada|CD:DR Congo|CF:Central African Republic|CG:Congo|CH:Switzerland|CI:Ivory Coast|CL:Chile|CM:Cameroon|CN:China|CO:Colombia|CR:Costa Rica|CU:Cuba|CV:Cape Verde|CY:Cyprus|CZ:Czechia|
DE:Germany|DJ:Djibouti|DK:Denmark|DM:Dominica|DO:Dominican Republic|DZ:Algeria|EC:Ecuador|EE:Estonia|EG:Egypt|ER:Eritrea|ES:Spain|ET:Ethiopia|
FI:Finland|FJ:Fiji|FR:France|GA:Gabon|GB:United Kingdom|GD:Grenada|GE:Georgia|GH:Ghana|GM:Gambia|GN:Guinea|GQ:Equatorial Guinea|GR:Greece|GT:Guatemala|GW:Guinea-Bissau|GY:Guyana|
HK:Hong Kong|HN:Honduras|HR:Croatia|HT:Haiti|HU:Hungary|ID:Indonesia|IE:Ireland|IL:Israel|IN:India|IQ:Iraq|IR:Iran|IS:Iceland|IT:Italy|
JM:Jamaica|JO:Jordan|JP:Japan|KE:Kenya|KG:Kyrgyzstan|KH:Cambodia|KI:Kiribati|KM:Comoros|KN:Saint Kitts and Nevis|KP:North Korea|KR:South Korea|KW:Kuwait|KZ:Kazakhstan|
LA:Laos|LB:Lebanon|LC:Saint Lucia|LI:Liechtenstein|LK:Sri Lanka|LR:Liberia|LS:Lesotho|LT:Lithuania|LU:Luxembourg|LV:Latvia|LY:Libya|
MA:Morocco|MC:Monaco|MD:Moldova|ME:Montenegro|MG:Madagascar|MK:North Macedonia|ML:Mali|MM:Myanmar|MN:Mongolia|MO:Macau|MR:Mauritania|MT:Malta|MU:Mauritius|MV:Maldives|MW:Malawi|MX:Mexico|MY:Malaysia|MZ:Mozambique|
NA:Namibia|NE:Niger|NG:Nigeria|NI:Nicaragua|NL:Netherlands|NO:Norway|NP:Nepal|NZ:New Zealand|OM:Oman|
PA:Panama|PE:Peru|PG:Papua New Guinea|PH:Philippines|PK:Pakistan|PL:Poland|PR:Puerto Rico|PS:Palestine|PT:Portugal|PY:Paraguay|QA:Qatar|
RO:Romania|RS:Serbia|RU:Russia|RW:Rwanda|SA:Saudi Arabia|SB:Solomon Islands|SC:Seychelles|SD:Sudan|SE:Sweden|SG:Singapore|SI:Slovenia|SK:Slovakia|SL:Sierra Leone|SM:San Marino|SN:Senegal|SO:Somalia|SR:Suriname|SS:South Sudan|SV:El Salvador|SY:Syria|SZ:Eswatini|
TD:Chad|TG:Togo|TH:Thailand|TJ:Tajikistan|TL:Timor-Leste|TM:Turkmenistan|TN:Tunisia|TO:Tonga|TR:Turkey|TT:Trinidad and Tobago|TW:Taiwan|TZ:Tanzania|
UA:Ukraine|UG:Uganda|US:United States|UY:Uruguay|UZ:Uzbekistan|VA:Vatican City|VC:Saint Vincent and the Grenadines|VE:Venezuela|VN:Vietnam|VU:Vanuatu|WS:Samoa|XK:Kosovo|YE:Yemen|ZA:South Africa|ZM:Zambia|ZW:Zimbabwe''';

/// Country names by code, e.g. `countryNames['DE'] == 'Germany'`.
final Map<String, String> countryNames = {
  for (final e in _table.replaceAll('\n', '').split('|'))
    if (e.contains(':')) e.substring(0, 2): e.substring(3),
};

/// Other names people put in a server name.
const _aliases = {
  'usa': 'US',
  'america': 'US',
  'uk': 'GB',
  'england': 'GB',
  'britain': 'GB',
  'great britain': 'GB',
  'holland': 'NL',
  'the netherlands': 'NL',
  'uae': 'AE',
  'emirates': 'AE',
  'korea': 'KR',
  'czech republic': 'CZ',
  'turkiye': 'TR',
  'türkiye': 'TR',
  'russian federation': 'RU',
  'viet nam': 'VN',
  'hongkong': 'HK',
  'deutschland': 'DE',
  'brasil': 'BR',
  'espana': 'ES',
  'españa': 'ES',
  'italia': 'IT',
  'suisse': 'CH',
  'schweiz': 'CH',
};

/// Big cities that name servers.
const _cities = {
  'frankfurt': 'DE',
  'berlin': 'DE',
  'munich': 'DE',
  'falkenstein': 'DE',
  'nuremberg': 'DE',
  'amsterdam': 'NL',
  'rotterdam': 'NL',
  'london': 'GB',
  'manchester': 'GB',
  'paris': 'FR',
  'marseille': 'FR',
  'madrid': 'ES',
  'barcelona': 'ES',
  'milan': 'IT',
  'rome': 'IT',
  'zurich': 'CH',
  'geneva': 'CH',
  'vienna': 'AT',
  'stockholm': 'SE',
  'oslo': 'NO',
  'copenhagen': 'DK',
  'helsinki': 'FI',
  'warsaw': 'PL',
  'prague': 'CZ',
  'budapest': 'HU',
  'bucharest': 'RO',
  'sofia': 'BG',
  'athens': 'GR',
  'istanbul': 'TR',
  'ankara': 'TR',
  'moscow': 'RU',
  'kyiv': 'UA',
  'kiev': 'UA',
  'tehran': 'IR',
  'dubai': 'AE',
  'tel aviv': 'IL',
  'mumbai': 'IN',
  'delhi': 'IN',
  'singapore': 'SG',
  'tokyo': 'JP',
  'osaka': 'JP',
  'seoul': 'KR',
  'taipei': 'TW',
  'sydney': 'AU',
  'melbourne': 'AU',
  'toronto': 'CA',
  'montreal': 'CA',
  'vancouver': 'CA',
  'new york': 'US',
  'nyc': 'US',
  'los angeles': 'US',
  'chicago': 'US',
  'dallas': 'US',
  'miami': 'US',
  'seattle': 'US',
  'ashburn': 'US',
  'san francisco': 'US',
  'sao paulo': 'BR',
  'johannesburg': 'ZA',
  'cairo': 'EG',
  'lisbon': 'PT',
  'dublin': 'IE',
  'brussels': 'BE',
};

/// Country-code domains people use as plain domains, so they say nothing
/// about where the server is.
const _genericTlds = {
  'io',
  'me',
  'co',
  'tv',
  'cc',
  'ai',
  'sh',
  'ly',
  'gg',
  'to',
  'fm',
  'am',
  'la',
  'ws',
  'xyz',
  'is',
  'in',
  'so',
  'ms',
  'ac',
  'do',
  'gl',
  'nu',
  'vc',
  'ag',
  'bz',
  'pw',
  'tk',
  'ml',
  'ga',
  'cf',
  'gq',
  'cx',
  'su',
  'eu',
  'ph',
  'ps',
  'ng',
};

final _flag = RegExp('[\u{1F1E6}-\u{1F1FF}]{2}', unicode: true);

/// The two-letter code of a flag emoji, or null.
String? codeOfFlag(String text) {
  final m = _flag.firstMatch(text);
  if (m == null) return null;
  final units = m.group(0)!.runes.map((r) => String.fromCharCode(r - 0x1F1E6 + 0x41)).join();
  return countryNames.containsKey(units) ? units : null;
}

/// The flag emoji of a country code.
String flagEmoji(String code) => String.fromCharCodes(code.toUpperCase().codeUnits.map((c) => c - 0x41 + 0x1F1E6));

/// Guesses the country of [p] from its name, then its address. Returns an
/// ISO code, or null when nothing in the profile tells.
String? guessCountry(ProfileInfo p) {
  if (p.protocol == 'tor') return null;
  final name = p.name;

  final flag = codeOfFlag(name);
  if (flag != null) return flag;

  // "[DE] fast", "(US)", "DE-Berlin-01", "NL 3": a code in brackets or at the start.
  final bracket = RegExp(r'[\[\(\{]([A-Za-z]{2})[\]\)\}]').firstMatch(name);
  if (bracket != null && countryNames.containsKey(bracket.group(1)!.toUpperCase())) {
    return bracket.group(1)!.toUpperCase();
  }
  final lead = RegExp(r'^\s*([A-Z]{2})(?![A-Za-z])').firstMatch(name);
  if (lead != null && countryNames.containsKey(lead.group(1))) return lead.group(1);

  final lower = name.toLowerCase();
  final byName = _byWord(lower, {
    for (final e in countryNames.entries) e.value.toLowerCase(): e.key,
    ..._aliases,
    ..._cities,
  });
  if (byName != null) return byName;

  // The address: de1.example.com, us-nyc.example.net, example.de.
  final host = p.server.toLowerCase();
  if (!RegExp(r'^[\d.:]+$').hasMatch(host)) {
    final first = RegExp(r'^([a-z]{2})(?:\d|-)').firstMatch(host);
    if (first != null && countryNames.containsKey(first.group(1)!.toUpperCase())) return first.group(1)!.toUpperCase();
    final tld = host.split('.').last;
    if (tld.length == 2 && !_genericTlds.contains(tld)) {
      final code = tld == 'uk' ? 'GB' : tld.toUpperCase();
      if (countryNames.containsKey(code)) return code;
    }
  }
  return null;
}

/// The key of the longest name from [names] that appears in [text] as whole words.
String? _byWord(String text, Map<String, String> names) {
  String? best;
  var bestLen = 0;
  for (final e in names.entries) {
    if (e.key.length <= bestLen) continue;
    final at = text.indexOf(e.key);
    if (at < 0) continue;
    final before = at == 0 || !_isLetter(text.codeUnitAt(at - 1));
    final end = at + e.key.length;
    final after = end >= text.length || !_isLetter(text.codeUnitAt(end));
    if (before && after) {
      best = e.value;
      bestLen = e.key.length;
    }
  }
  return best;
}

bool _isLetter(int c) => (c >= 0x61 && c <= 0x7A) || c > 0x7F;
