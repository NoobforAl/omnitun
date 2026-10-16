// Generated from Natural Earth (public domain): the land grid of the globe and the centre of each
// country. Not edited by hand.

/// Land (1) or sea (0) for every 1.5-degree cell, row by row from 90N, as packed bits (base64).
const landGridWidth = 240;
const landGridHeight = 120;
const landGridStepDegrees = 1.5;
const landGridBase64 =
    'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA'
    'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAfwAAf/wAAAAAAAAAAAAAA'
    'AAAAAAAAAAAAAAAAAA///////44AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAP//n/////gAAA+AAQAAAAPAAAAAAAAAAAAAAAAA'
    'O3/8/////+AAAfgAAAAAAAB8AAAAAAAAAAAAAADAAMfwf////+AAADAAAAAAAAAOAAAAAAAAAAAAAAA/jr/AAP///+AAAAAA'
    'AAPwAA//8AAPAAAAAAAAAAfAAOaAAD///+AAAAAAAA4AAH//wAAAAAAAAAAAAA/+zs/4AB///4AAAAAAABwDA/////wHAAAA'
    'gAGAAAZ/ws//AB///4AAAAAAABwHf/////7/+AAAAB//wP8/8+Hv4A///wAAAA/8AAAH/////////+g8wH//////8/vF8Af/'
    '+AAAAD//xj//v////////////D////////+B/g//gAAAAH//5///////////////OP////////8P5Af4AP4AAP7+H///////'
    '////////AA////////PA+Af4ADgAA/3////////////////+AD///////8COIAHwAAAAD/P////////////////8AD//////'
    '/4APwADgAAAAH/P//////////////w/gAA/QH////4APzAAAAAAAD/D/////////////5hgAAAHAA////8AP/gAAAAAMAOD/'
    '///////////8AHgAAAYAAP////wH/gAAAAAOBuP////////////wAPgAAAAAAH////+f/8AAAAA2Ah/////////////gAPAA'
    'AAAAAT////+f/+AAAAA3H///////////////AOAAAAAAAB/////f/+AAAAAnv///////////////AIAAAAAAAB///////2AA'
    'AAAE///////////////9AAAAAAAAAA//////+HgAAAAH///////////////5AAAAAAAAAAf/////+BgAAAAD////////////'
    '///7AAAAAAAAAAf//////wAAAAAB///zfx/////////wAAAAAAAAAAf/////6AAAAAAB/7/gPj/////////jgAAAAAAAAAf/'
    '////gAAAAAA/wc/gHx////////8HAAAAAAAAAAf/////AAAAAAA/wGf/n4////////4EAAAAAAAAAAf////+AAAAAAA/Bic/'
    '/5///////3gGAAAAAAAAAAP////8AAAAAAA/ACM//4///////hwEAAAAAAAAAAP////4AAAAAAAefgA//////////4wcAAAA'
    'AAAAAAH////4AAAAAAAP/gAA/////////wz8AAAAAAAAAAB////wAAAAAAAf/gAA/////////4HgAAAAAAAAAAA////AAAAA'
    'AAA//8cB/////////4EAAAAAAAAAAAA////AAAAAAAB///f//////////4AAAAAAAAAAAAAX/5DAAAAAAAB//////z//////'
    '/4AAAAAAAAAAAAAL/gBgAAAAAAH//////7///////4AAAAAAAAAAAAAF/gBgAAAAAAP////+/8P//////wAAAAAAAAAAAAAE'
    '/gAAAAAAAAP////+f+wH/////gAAAAAAAAAAAAACfgBAAAAAAAf/////P/4D/////IAAAAAAAAAAAAAAPgAYAAAAAAf/////'
    'v/8D//f/4AAAAAAAAAAAAAAAPwwMAAAAAAf/////n/4A/8P+QAAAAAAAAAAAAAAAP5wBwAAAAAf/////n/wA/wH+wAAAAAAA'
    'AAAAAAAAB/gAAAAAAAf/////z/gAfgH+AMAAAAAAAAAAAAAAAT8AAAAAAAf/////7+AAfAF/AIAAAAAAAAAAAAAAAB+AAAAA'
    'AA//////94AAPAB/gMAAAAAAAAAAAAAAAAMAAAAAAAf//////AAAPAB/gIAAAAAAAAAAAAAAAAEDwAAAAAP/////+MAAOAAn'
    'gFAAAAAAAAAAAAAAAAGH/gAAAAH//////8AAGAACAAAAAAAAAAAAAAAAAABv/wAAAAH//////8AAFAAgADAAAAAAAAAAAAAA'
    'AAAP/8AAAAD//////4AABgAwABAAAAAAAAAAAAAAAAAP//gAAAB/H////wAAAACYBgAAAAAAAAAAAAAAAAAP//wAAAAAA///'
    '/wAAAADYDgAAAAAAAAAAAAAAAAAf//wAAAAAA////gAAAAB4PgAAAAAAAAAAAAAAAAAf//4AAAAAA///+AAAAAA4foQAAAAA'
    'AAAAAAAAAAA///+AAAAAA///8AAAAAAYfACAAAAAAAAAAAAAAAA////gAAAAA///4AAAAAAcfYDcAAAAAAAAAAAAAAA////8'
    'AAAAAf//4AAAAAAOCcB/BAAAAAAAAAAAAAA/////AAAAAP//wAAAAAAGAEAPiAAAAAAAAAAAAAA/////gAAAAP//wAAAAAAD'
    '4AAPwAAAAAAAAAAAAAAf////gAAAAH//wAAAAAAAOAAPYAAAAAAAAAAAAAAP////AAAAAH//4AAAAAAAACAAMAAAAAAAAAAA'
    'AAAP///+AAAAAH//4AAAAAAAAAAAAAAAAAAAAAAAAAAH///+AAAAAH//4IAAAAAAAAHjAAAAAAAAAAAAAAAH///8AAAAAP//'
    '4MAAAAAAAAvjAAAAAAAAAAAAAAAD///8AAAAAP//4cAAAAAAAB/jgAEAAAAAAAAAAAAA///8AAAAAP//x4AAAAAAAD/7gAAA'
    'AAAAAAAAAAAAf//8AAAAAP//B4AAAAAAAH//wAAAAAAAAAAAAAAAf//4AAAAAH/+B4AAAAAAAf//4AAAAAAAAAAAAAAAf//4'
    'AAAAAH//BwAAAAAAD///8AIAAAAAAAAAAAAAf//gAAAAAH//BwAAAAAAD///+AAAAAAAAAAAAAAAf/+AAAAAAD/+BwAAAAAA'
    'D////AAAAAAAAAAAAAAAf/8AAAAAAD/8AAAAAAAAD////AAAAAAAAAAAAAAA//8AAAAAAD/8AAAAAAAAD////AAAAAAAAAAA'
    'AAAA//4AAAAAAB/4AAAAAAAAD////AAAAAAAAAAAAAAA//wAAAAAAA/wAAAAAAAAB////AAAAAAAAAAAAAAA//gAAAAAAA/g'
    'AAAAAAAAB/h//AAAAAAAAAAAAAAA//gAAAAAAA/AAAAAAAAAB+A/+AAAAAAAAAAAAAAA/8AAAAAAAAAAAAAAAAAAAAAP+AAQ'
    'AAAAAAAAAAAB/8AAAAAAAAAAAAAAAAAAAAAH8AAIAAAAAAAAAAAB/4AAAAAAAAAAAAAAAAAAAAADwAAOAAAAAAAAAAAB/gAA'
    'AAAAAAAAAAAAAAAAAAAAAAAMAAAAAAAAAAAB+AAAAAAAAAAAAAAAAAAAAAAA4AAYAAAAAAAAAAAD+AAAAAAAAAAAAAAAAAAA'
    'AAAAYAAwAAAAAAAAAAAB+AAAAAAAAAAAAAAAAAAAAAAAAADgAAAAAAAAAAAD4AAAAAAAAAAAAAAAAAAAAAAAAAHAAAAAAAAA'
    'AAAD8AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAH4AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADwAAAAAAAAAAA'
    'AAAAAAAAAAAAAAAAAAAAAAAAAAADwIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAB4AAAAAAAAAAAAAAAAAAAAAAAAAAA'
    'AAAAAAAAAAAA8AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA'
    'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA'
    'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA'
    'AAAABgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHAAAAAAAAAAAAH4AACAefH/vgAAAAAAAAAAAAAAAGAAAAAAAAAAA'
    'B//8B///////4AAAAAAAAAAAAAAA/AAAAAAAABgP///8f////////8AAAAAAAAAAAAAB/gAAAABbf//////8///////////A'
    'AAAAAAAAHggAvgAAAAH////////////////////AAAAAAjv4D////gAAAAf///////////////////4AAAAB////////8AAA'
    'AD////////////////////gAAAHf///////8AAAAH/////////////////////gAABj////////wAAHg////////////////'
    '//////4AAAAD///////+Bw/gD////////////////////+AAAAB//////////APf//////////////////////AAAAA/////'
    '//////////////////////////////8A/+AB////////////////////////////////////////////////////////////'
    '////////////////////////////////////////////////////////////////////////////////////////////////'
    ;

/// (latitude, longitude) of the middle of each country, by ISO 3166-1 alpha-2 code.
const countryCenters = <String, (double, double)>{
  'AD': (42.55, 1.6),
  'AE': (23.9, 54.2),
  'AF': (33.9, 66.1),
  'AG': (17.06, -61.8),
  'AL': (41.1, 20.0),
  'AM': (40.2, 45.0),
  'AO': (-12.3, 17.5),
  'AQ': (-80.5, 21.3),
  'AR': (-35.2, -65.1),
  'AT': (47.6, 14.1),
  'AU': (-25.6, 134.4),
  'AZ': (40.3, 47.7),
  'BA': (44.2, 17.8),
  'BB': (13.19, -59.54),
  'BD': (23.8, 90.3),
  'BE': (50.7, 4.6),
  'BF': (12.3, -1.8),
  'BG': (42.8, 25.2),
  'BH': (26.07, 50.55),
  'BI': (-3.4, 29.9),
  'BJ': (9.6, 2.3),
  'BN': (4.7, 114.9),
  'BO': (-16.7, -64.6),
  'BR': (-10.8, -53.1),
  'BS': (24.5, -77.9),
  'BT': (27.4, 90.5),
  'BW': (-22.1, 23.8),
  'BY': (53.5, 28.0),
  'BZ': (17.2, -88.7),
  'CA': (57.7, -101.6),
  'CD': (-2.9, 23.6),
  'CF': (6.5, 20.4),
  'CG': (-0.8, 15.1),
  'CH': (46.8, 8.1),
  'CI': (7.6, -5.6),
  'CL': (-37.3, -71.7),
  'CM': (5.7, 12.6),
  'CN': (36.6, 103.9),
  'CN-TW': (23.7, 121.0),
  'CO': (3.9, -73.1),
  'CR': (10.0, -84.2),
  'CU': (21.6, -79.0),
  'CV': (16.0, -24.0),
  'CY': (34.9, 33.0),
  'CZ': (49.8, 15.3),
  'DE': (51.1, 10.3),
  'DJ': (11.8, 42.5),
  'DK': (56.2, 9.3),
  'DM': (15.4, -61.37),
  'DO': (18.9, -70.5),
  'DZ': (28.2, 2.6),
  'EC': (-1.5, -78.4),
  'EE': (58.6, 25.8),
  'EG': (26.5, 29.8),
  'EH': (24.3, -12.1),
  'ER': (15.4, 38.7),
  'ES': (40.3, -3.6),
  'ET': (8.7, 39.6),
  'FI': (64.5, 26.2),
  'FJ': (-17.8, 178.0),
  'FK': (-51.7, -59.4),
  'FR': (46.6, 2.3),
  'GA': (-0.6, 11.7),
  'GB': (53.9, -2.7),
  'GD': (12.1, -61.7),
  'GE': (42.2, 43.5),
  'GH': (7.9, -1.2),
  'GL': (74.8, -41.5),
  'GM': (13.5, -15.4),
  'GN': (10.4, -11.1),
  'GQ': (1.6, 10.4),
  'GR': (39.3, 22.6),
  'GT': (15.7, -90.4),
  'GW': (12.0, -15.1),
  'GY': (4.8, -59.0),
  'HK': (22.3, 114.17),
  'HN': (14.8, -86.6),
  'HR': (45.0, 16.6),
  'HT': (18.9, -72.7),
  'HU': (47.2, 19.4),
  'ID': (-0.3, 114.0),
  'IE': (53.2, -8.0),
  'IL': (31.5, 35.0),
  'IN': (22.9, 79.6),
  'IQ': (33.0, 43.8),
  'IR': (32.5, 54.3),
  'IS': (65.1, -18.8),
  'IT': (43.5, 12.2),
  'JM': (18.1, -77.3),
  'JO': (31.2, 36.8),
  'JP': (36.0, 136.9),
  'KE': (0.6, 37.8),
  'KG': (41.5, 74.6),
  'KH': (12.7, 104.9),
  'KI': (1.45, 173.0),
  'KM': (-11.7, 43.3),
  'KN': (17.3, -62.7),
  'KP': (40.1, 127.2),
  'KR': (36.4, 127.8),
  'KW': (29.3, 47.6),
  'KZ': (48.2, 67.3),
  'LA': (18.4, 103.8),
  'LB': (33.9, 35.9),
  'LC': (13.9, -60.98),
  'LI': (47.16, 9.55),
  'LK': (7.7, 80.7),
  'LR': (6.4, -9.4),
  'LS': (-29.6, 28.2),
  'LT': (55.3, 23.9),
  'LU': (49.8, 6.0),
  'LV': (56.8, 24.8),
  'LY': (27.0, 18.0),
  'MA': (29.9, -8.4),
  'MC': (43.74, 7.42),
  'MD': (47.2, 28.4),
  'ME': (42.8, 19.3),
  'MG': (-19.4, 46.7),
  'MK': (41.6, 21.7),
  'ML': (17.3, -3.5),
  'MM': (21.0, 96.5),
  'MN': (46.8, 102.9),
  'MO': (22.2, 113.55),
  'MR': (20.2, -10.3),
  'MT': (35.9, 14.4),
  'MU': (-20.3, 57.6),
  'MV': (3.2, 73.2),
  'MW': (-13.2, 34.2),
  'MX': (23.9, -102.6),
  'MY': (3.5, 114.7),
  'MZ': (-17.2, 35.5),
  'NA': (-22.1, 17.2),
  'NC': (-21.3, 165.5),
  'NE': (17.3, 9.3),
  'NG': (9.5, 8.0),
  'NI': (12.8, -85.0),
  'NL': (52.3, 5.5),
  'NO': (64.5, 14.2),
  'NP': (28.2, 84.0),
  'NZ': (-44.0, 170.5),
  'OM': (20.6, 56.1),
  'PA': (8.5, -80.1),
  'PE': (-9.2, -74.4),
  'PG': (-6.6, 144.3),
  'PH': (15.8, 121.5),
  'PK': (30.0, 69.4),
  'PL': (52.1, 19.3),
  'PR': (18.2, -66.5),
  'PS': (31.9, 35.3),
  'PT': (39.6, -8.1),
  'PY': (-23.2, -58.4),
  'QA': (25.3, 51.2),
  'RO': (45.9, 24.9),
  'RS': (44.2, 20.8),
  'RU': (61.7, 99.2),
  'RW': (-2.0, 29.9),
  'SA': (24.1, 44.5),
  'SB': (-7.9, 159.1),
  'SC': (-4.68, 55.49),
  'SD': (16.0, 29.9),
  'SE': (62.8, 16.6),
  'SG': (1.35, 103.82),
  'SI': (46.1, 14.9),
  'SK': (48.7, 19.5),
  'SL': (8.5, -11.8),
  'SM': (43.94, 12.46),
  'SN': (14.4, -14.5),
  'SO': (4.8, 45.7),
  'SR': (4.1, -55.9),
  'SS': (7.3, 30.2),
  'SV': (13.7, -88.9),
  'SY': (35.0, 38.5),
  'SZ': (-26.5, 31.4),
  'TD': (15.3, 18.6),
  'TF': (-49.3, 69.5),
  'TG': (8.4, 1.0),
  'TH': (15.0, 101.0),
  'TJ': (38.6, 71.0),
  'TL': (-8.8, 126.0),
  'TM': (39.1, 59.3),
  'TN': (34.2, 9.5),
  'TO': (-21.18, -175.2),
  'TR': (39.0, 35.4),
  'TT': (10.4, -61.3),
  'TW': (23.7, 121.0),
  'TZ': (-6.3, 34.8),
  'UA': (49.1, 31.2),
  'UG': (1.3, 32.4),
  'US': (39.5, -99.1),
  'UY': (-32.8, -56.0),
  'UZ': (41.7, 63.2),
  'VA': (41.9, 12.45),
  'VC': (13.25, -61.2),
  'VE': (7.2, -66.2),
  'VN': (16.7, 106.3),
  'VU': (-15.2, 166.9),
  'WS': (-13.76, -172.1),
  'XK': (42.6, 20.9),
  'YE': (15.9, 47.5),
  'ZA': (-29.0, 25.1),
  'ZM': (-13.4, 27.7),
  'ZW': (-18.9, 29.8),
};
