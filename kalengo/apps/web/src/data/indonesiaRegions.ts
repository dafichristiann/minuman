export interface DistrictInfo {
  readonly name: string;
  readonly postalCode: string;
}

export interface CityInfo {
  readonly name: string;
  readonly districts: readonly DistrictInfo[];
}

export interface ProvinceInfo {
  readonly name: string;
  readonly cities: readonly CityInfo[];
}

export const INDONESIA_REGIONS: readonly ProvinceInfo[] = [
  {
    name: "DKI Jakarta",
    cities: [
      {
        name: "Kota Jakarta Pusat",
        districts: [
          { name: "Gambir", postalCode: "10110" },
          { name: "Menteng", postalCode: "10310" },
          { name: "Senen", postalCode: "10410" },
          { name: "Cempaka Putih", postalCode: "10510" },
          { name: "Kemayoran", postalCode: "10620" },
          { name: "Tanah Abang", postalCode: "10250" },
          { name: "Sawah Besar", postalCode: "10710" },
          { name: "Johar Baru", postalCode: "10560" },
        ],
      },
      {
        name: "Kota Jakarta Selatan",
        districts: [
          { name: "Kebayoran Baru", postalCode: "12110" },
          { name: "Kebayoran Lama", postalCode: "12240" },
          { name: "Pesanggrahan", postalCode: "12320" },
          { name: "Cilandak", postalCode: "12430" },
          { name: "Pasar Minggu", postalCode: "12520" },
          { name: "Jagakarsa", postalCode: "12620" },
          { name: "Mampang Prapatan", postalCode: "12790" },
          { name: "Pancoran", postalCode: "12780" },
          { name: "Tebet", postalCode: "12810" },
          { name: "Setiabudi", postalCode: "12910" },
        ],
      },
      {
        name: "Kota Jakarta Barat",
        districts: [
          { name: "Kebon Jeruk", postalCode: "11530" },
          { name: "Grogol Petamburan", postalCode: "11450" },
          { name: "Palmerah", postalCode: "11480" },
          { name: "Kembangan", postalCode: "11610" },
          { name: "Cengkareng", postalCode: "11730" },
          { name: "Kalideres", postalCode: "11840" },
          { name: "Tambora", postalCode: "11220" },
          { name: "Taman Sari", postalCode: "11150" },
        ],
      },
      {
        name: "Kota Jakarta Timur",
        districts: [
          { name: "Matraman", postalCode: "13140" },
          { name: "Pulo Gadung", postalCode: "13220" },
          { name: "Jatinegara", postalCode: "13310" },
          { name: "Duren Sawit", postalCode: "13440" },
          { name: "Kramat Jati", postalCode: "13510" },
          { name: "Makasar", postalCode: "13570" },
          { name: "Pasar Rebo", postalCode: "13760" },
          { name: "Ciracas", postalCode: "13740" },
          { name: "Cipayung", postalCode: "13840" },
          { name: "Cakung", postalCode: "13910" },
        ],
      },
      {
        name: "Kota Jakarta Utara",
        districts: [
          { name: "Penjaringan", postalCode: "14450" },
          { name: "Tanjung Priok", postalCode: "14310" },
          { name: "Koja", postalCode: "14210" },
          { name: "Cilincing", postalCode: "14120" },
          { name: "Pademangan", postalCode: "14420" },
          { name: "Kelapa Gading", postalCode: "14240" },
        ],
      },
    ],
  },
  {
    name: "Jawa Barat",
    cities: [
      {
        name: "Kota Bandung",
        districts: [
          { name: "Coblong", postalCode: "40132" },
          { name: "Sukasari", postalCode: "40152" },
          { name: "Sumur Bandung", postalCode: "40111" },
          { name: "Cicendo", postalCode: "40173" },
          { name: "Lengkong", postalCode: "40264" },
          { name: "Batununggal", postalCode: "40272" },
          { name: "Buahbatu", postalCode: "40286" },
          { name: "Cidadap", postalCode: "40142" },
        ],
      },
      {
        name: "Kab. Bandung",
        districts: [
          { name: "Soreang", postalCode: "40911" },
          { name: "Banjaran", postalCode: "40377" },
          { name: "Baleendah", postalCode: "40375" },
          { name: "Dayeuhkolot", postalCode: "40258" },
          { name: "Cileunyi", postalCode: "40622" },
        ],
      },
      {
        name: "Kota Bogor",
        districts: [
          { name: "Bogor Tengah", postalCode: "16121" },
          { name: "Bogor Timur", postalCode: "16143" },
          { name: "Bogor Utara", postalCode: "16153" },
          { name: "Bogor Selatan", postalCode: "16132" },
          { name: "Bogor Barat", postalCode: "16113" },
          { name: "Tanah Sareal", postalCode: "16161" },
        ],
      },
      {
        name: "Kab. Bogor",
        districts: [
          { name: "Cibinong", postalCode: "16911" },
          { name: "Ciawi", postalCode: "16720" },
          { name: "Babakan Madang", postalCode: "16810" },
          { name: "Cileungsi", postalCode: "16820" },
        ],
      },
      {
        name: "Kota Bekasi",
        districts: [
          { name: "Bekasi Timur", postalCode: "17113" },
          { name: "Bekasi Barat", postalCode: "17145" },
          { name: "Bekasi Selatan", postalCode: "17148" },
          { name: "Bekasi Utara", postalCode: "17124" },
          { name: "Pondok Gede", postalCode: "17411" },
        ],
      },
      {
        name: "Kab. Bekasi",
        districts: [
          { name: "Cikarang Pusat", postalCode: "17530" },
          { name: "Cikarang Selatan", postalCode: "17550" },
          { name: "Cikarang Utara", postalCode: "17534" },
          { name: "Tambun Selatan", postalCode: "17510" },
        ],
      },
      {
        name: "Kota Depok",
        districts: [
          { name: "Pancoran Mas", postalCode: "16436" },
          { name: "Sukmajaya", postalCode: "16412" },
          { name: "Beji", postalCode: "16424" },
          { name: "Cinere", postalCode: "16514" },
          { name: "Cimanggis", postalCode: "16452" },
        ],
      },
      {
        name: "Kota Cimahi",
        districts: [
          { name: "Cimahi Selatan", postalCode: "40532" },
          { name: "Cimahi Tengah", postalCode: "40525" },
          { name: "Cimahi Utara", postalCode: "40512" },
        ],
      },
      {
        name: "Kota Cirebon",
        districts: [
          { name: "Kejaksan", postalCode: "45123" },
          { name: "Kesambi", postalCode: "45134" },
          { name: "Lemahwungkuk", postalCode: "45111" },
        ],
      },
      {
        name: "Kab. Cirebon",
        districts: [
          { name: "Sumber", postalCode: "45611" },
          { name: "Kedawung", postalCode: "45153" },
          { name: "Weru", postalCode: "45154" },
        ],
      },
      {
        name: "Kota Sukabumi",
        districts: [
          { name: "Cikole", postalCode: "43111" },
          { name: "Citamiang", postalCode: "43141" },
          { name: "Warudoyong", postalCode: "43131" },
        ],
      },
      {
        name: "Kab. Sukabumi",
        districts: [
          { name: "Palabuhanratu", postalCode: "43364" },
          { name: "Cibadak", postalCode: "43351" },
          { name: "Cisaat", postalCode: "43152" },
        ],
      },
      {
        name: "Kab. Garut",
        districts: [
          { name: "Garut Kota", postalCode: "44118" },
          { name: "Tarogong Kidul", postalCode: "44151" },
          { name: "Tarogong Kaler", postalCode: "44151" },
        ],
      },
      {
        name: "Kota Tasikmalaya",
        districts: [
          { name: "Cihideung", postalCode: "46122" },
          { name: "Tawang", postalCode: "46112" },
          { name: "Cipedes", postalCode: "46132" },
        ],
      },
      {
        name: "Kab. Karawang",
        districts: [
          { name: "Karawang Barat", postalCode: "41311" },
          { name: "Karawang Timur", postalCode: "41314" },
          { name: "Telukjambe Timur", postalCode: "41361" },
        ],
      },
    ],
  },
  {
    name: "Banten",
    cities: [
      {
        name: "Kota Tangerang",
        districts: [
          { name: "Tangerang", postalCode: "15111" },
          { name: "Cipondoh", postalCode: "15148" },
          { name: "Ciledug", postalCode: "15153" },
          { name: "Karawaci", postalCode: "15115" },
          { name: "Batuceper", postalCode: "15122" },
        ],
      },
      {
        name: "Kota Tangerang Selatan",
        districts: [
          { name: "Serpong", postalCode: "15310" },
          { name: "Serpong Utara", postalCode: "15326" },
          { name: "Pondok Aren", postalCode: "15224" },
          { name: "Ciputat", postalCode: "15411" },
          { name: "Pamulang", postalCode: "15417" },
        ],
      },
      {
        name: "Kab. Tangerang",
        districts: [
          { name: "Tigaraksa", postalCode: "15720" },
          { name: "Kelapa Dua", postalCode: "15810" },
          { name: "Curug", postalCode: "15810" },
          { name: "Balaraja", postalCode: "15610" },
        ],
      },
      {
        name: "Kota Serang",
        districts: [
          { name: "Serang", postalCode: "42111" },
          { name: "Cipocok Jaya", postalCode: "42121" },
          { name: "Kasemen", postalCode: "42191" },
        ],
      },
      {
        name: "Kota Cilegon",
        districts: [
          { name: "Cibeber", postalCode: "42422" },
          { name: "Cilegon", postalCode: "42416" },
          { name: "Jombang", postalCode: "42411" },
        ],
      },
    ],
  },
  {
    name: "Jawa Tengah",
    cities: [
      {
        name: "Kota Semarang",
        districts: [
          { name: "Semarang Tengah", postalCode: "50132" },
          { name: "Semarang Barat", postalCode: "50148" },
          { name: "Semarang Selatan", postalCode: "50249" },
          { name: "Banyumanik", postalCode: "50269" },
          { name: "Pedurungan", postalCode: "50192" },
        ],
      },
      {
        name: "Kota Surakarta",
        districts: [
          { name: "Banjarsari", postalCode: "57139" },
          { name: "Jebres", postalCode: "57126" },
          { name: "Laweyan", postalCode: "57148" },
          { name: "Pasar Kliwon", postalCode: "57118" },
          { name: "Serengan", postalCode: "57155" },
        ],
      },
      {
        name: "Kota Magelang",
        districts: [
          { name: "Magelang Tengah", postalCode: "56117" },
          { name: "Magelang Selatan", postalCode: "56126" },
          { name: "Magelang Utara", postalCode: "56115" },
        ],
      },
      {
        name: "Kab. Banyumas",
        districts: [
          { name: "Purwokerto Barat", postalCode: "53131" },
          { name: "Purwokerto Timur", postalCode: "53114" },
          { name: "Purwokerto Selatan", postalCode: "53146" },
          { name: "Purwokerto Utara", postalCode: "53127" },
        ],
      },
      {
        name: "Kab. Kudus",
        districts: [
          { name: "Kota Kudus", postalCode: "59313" },
          { name: "Jati", postalCode: "59349" },
          { name: "Bae", postalCode: "59381" },
        ],
      },
    ],
  },
  {
    name: "DI Yogyakarta",
    cities: [
      {
        name: "Kota Yogyakarta",
        districts: [
          { name: "Gondokusuman", postalCode: "55224" },
          { name: "Danurejan", postalCode: "55213" },
          { name: "Kraton", postalCode: "55132" },
          { name: "Mantrijeron", postalCode: "55143" },
          { name: "Umbulharjo", postalCode: "55167" },
          { name: "Jetis", postalCode: "55233" },
          { name: "Malioboro / Gedongtengen", postalCode: "55271" },
        ],
      },
      {
        name: "Kab. Sleman",
        districts: [
          { name: "Depok", postalCode: "55281" },
          { name: "Mlati", postalCode: "55284" },
          { name: "Ngaglik", postalCode: "55581" },
          { name: "Gamping", postalCode: "55294" },
          { name: "Kalasan", postalCode: "55571" },
        ],
      },
      {
        name: "Kab. Bantul",
        districts: [
          { name: "Bantul", postalCode: "55711" },
          { name: "Banguntapan", postalCode: "55198" },
          { name: "Sewon", postalCode: "55188" },
          { name: "Kasihan", postalCode: "55183" },
        ],
      },
      {
        name: "Kab. Gunungkidul",
        districts: [
          { name: "Wonosari", postalCode: "55813" },
          { name: "Playen", postalCode: "55861" },
          { name: "Semanu", postalCode: "55893" },
        ],
      },
      {
        name: "Kab. Kulon Progo",
        districts: [
          { name: "Wates", postalCode: "55611" },
          { name: "Pengasih", postalCode: "55652" },
          { name: "Sentolo", postalCode: "55664" },
        ],
      },
    ],
  },
  {
    name: "Jawa Timur",
    cities: [
      {
        name: "Kota Surabaya",
        districts: [
          { name: "Gubeng", postalCode: "60281" },
          { name: "Wonokromo", postalCode: "60243" },
          { name: "Tegalsari", postalCode: "60262" },
          { name: "Genteng", postalCode: "60272" },
          { name: "Rungkut", postalCode: "60293" },
          { name: "Sukolilo", postalCode: "60111" },
          { name: "Kenjeran", postalCode: "60129" },
        ],
      },
      {
        name: "Kota Malang",
        districts: [
          { name: "Klojen", postalCode: "65111" },
          { name: "Lowokwaru", postalCode: "65141" },
          { name: "Blimbing", postalCode: "65126" },
          { name: "Sukun", postalCode: "65147" },
          { name: "Kedungkandang", postalCode: "65137" },
        ],
      },
      {
        name: "Kota Batu",
        districts: [
          { name: "Batu", postalCode: "65311" },
          { name: "Bumiaji", postalCode: "65331" },
          { name: "Junrejo", postalCode: "65321" },
        ],
      },
      {
        name: "Kab. Sidoarjo",
        districts: [
          { name: "Sidoarjo", postalCode: "61218" },
          { name: "Waru", postalCode: "61256" },
          { name: "Candi", postalCode: "61271" },
          { name: "Gedangan", postalCode: "61254" },
        ],
      },
      {
        name: "Kab. Gresik",
        districts: [
          { name: "Gresik", postalCode: "61114" },
          { name: "Kebomas", postalCode: "61124" },
          { name: "Manyar", postalCode: "61151" },
        ],
      },
    ],
  },
  {
    name: "Bali",
    cities: [
      {
        name: "Kota Denpasar",
        districts: [
          { name: "Denpasar Selatan", postalCode: "80227" },
          { name: "Denpasar Barat", postalCode: "80119" },
          { name: "Denpasar Utara", postalCode: "80115" },
          { name: "Denpasar Timur", postalCode: "80237" },
        ],
      },
      {
        name: "Kab. Badung",
        districts: [
          { name: "Kuta", postalCode: "80361" },
          { name: "Kuta Selatan", postalCode: "80363" },
          { name: "Kuta Utara", postalCode: "80361" },
          { name: "Mengwi", postalCode: "80351" },
        ],
      },
      {
        name: "Kab. Gianyar",
        districts: [
          { name: "Gianyar", postalCode: "80511" },
          { name: "Ubud", postalCode: "80571" },
          { name: "Sukawati", postalCode: "80582" },
        ],
      },
      {
        name: "Kab. Tabanan",
        districts: [
          { name: "Tabanan", postalCode: "82113" },
          { name: "Kediri", postalCode: "82121" },
          { name: "Baturiti", postalCode: "82191" },
        ],
      },
      {
        name: "Kab. Buleleng",
        districts: [
          { name: "Buleleng", postalCode: "81119" },
          { name: "Sukasada", postalCode: "81161" },
          { name: "Seririt", postalCode: "81153" },
        ],
      },
    ],
  },
  {
    name: "Sumatera Utara",
    cities: [
      {
        name: "Kota Medan",
        districts: [
          { name: "Medan Kota", postalCode: "20212" },
          { name: "Medan Baru", postalCode: "20153" },
          { name: "Medan Petisah", postalCode: "20112" },
          { name: "Medan Helvetia", postalCode: "20124" },
          { name: "Medan Sunggal", postalCode: "20122" },
        ],
      },
      {
        name: "Kota Binjai",
        districts: [
          { name: "Binjai Kota", postalCode: "20711" },
          { name: "Binjai Barat", postalCode: "20719" },
          { name: "Binjai Timur", postalCode: "20731" },
        ],
      },
      {
        name: "Kota Pematangsiantar",
        districts: [
          { name: "Siantar Barat", postalCode: "21111" },
          { name: "Siantar Timur", postalCode: "21132" },
          { name: "Siantar Selatan", postalCode: "21121" },
        ],
      },
      {
        name: "Kab. Deli Serdang",
        districts: [
          { name: "Lubuk Pakam", postalCode: "20511" },
          { name: "Percut Sei Tuan", postalCode: "20371" },
          { name: "Sunggal", postalCode: "20351" },
        ],
      },
    ],
  },
  {
    name: "Sumatera Barat",
    cities: [
      {
        name: "Kota Padang",
        districts: [
          { name: "Padang Barat", postalCode: "25114" },
          { name: "Padang Timur", postalCode: "25121" },
          { name: "Padang Utara", postalCode: "25133" },
          { name: "Koto Tangah", postalCode: "25171" },
        ],
      },
      {
        name: "Kota Bukittinggi",
        districts: [
          { name: "Guguk Panjang", postalCode: "26113" },
          { name: "Mandiangin Koto Selayan", postalCode: "26122" },
          { name: "Aur Birugo Tigo Baleh", postalCode: "26131" },
        ],
      },
    ],
  },
  {
    name: "Riau",
    cities: [
      {
        name: "Kota Pekanbaru",
        districts: [
          { name: "Pekanbaru Kota", postalCode: "28111" },
          { name: "Sukajadi", postalCode: "28124" },
          { name: "Tampan / Tuah Madani", postalCode: "28293" },
          { name: "Marpoyan Damai", postalCode: "28282" },
        ],
      },
      {
        name: "Kota Dumai",
        districts: [
          { name: "Dumai Kota", postalCode: "28811" },
          { name: "Dumai Barat", postalCode: "28821" },
          { name: "Dumai Timur", postalCode: "28814" },
        ],
      },
    ],
  },
  {
    name: "Sumatera Selatan",
    cities: [
      {
        name: "Kota Palembang",
        districts: [
          { name: "Ilir Barat I", postalCode: "30139" },
          { name: "Ilir Timur I", postalCode: "30125" },
          { name: "Sukarami", postalCode: "30151" },
          { name: "Seberang Ulu I", postalCode: "30252" },
          { name: "Alang-Alang Lebar", postalCode: "30154" },
        ],
      },
    ],
  },
  {
    name: "Kalimantan Timur",
    cities: [
      {
        name: "Kota Samarinda",
        districts: [
          { name: "Samarinda Kota", postalCode: "75111" },
          { name: "Samarinda Ulu", postalCode: "75123" },
          { name: "Samarinda Seberang", postalCode: "75131" },
          { name: "Sungai Kunjang", postalCode: "75126" },
        ],
      },
      {
        name: "Kota Balikpapan",
        districts: [
          { name: "Balikpapan Kota", postalCode: "76111" },
          { name: "Balikpapan Selatan", postalCode: "76114" },
          { name: "Balikpapan Utara", postalCode: "76125" },
          { name: "Balikpapan Barat", postalCode: "76131" },
        ],
      },
    ],
  },
  {
    name: "Sulawesi Selatan",
    cities: [
      {
        name: "Kota Makassar",
        districts: [
          { name: "Ujung Pandang", postalCode: "90111" },
          { name: "Panakkukang", postalCode: "90231" },
          { name: "Rappocini", postalCode: "90222" },
          { name: "Tamalanrea", postalCode: "90245" },
          { name: "Bontoala", postalCode: "90151" },
        ],
      },
    ],
  },
];

export function getProvinces(): string[] {
  return INDONESIA_REGIONS.map((r) => r.name);
}

export function getCitiesByProvince(provinceName: string): string[] {
  if (!provinceName) return [];
  const normalized = provinceName.trim().toLowerCase();
  const prov = INDONESIA_REGIONS.find((p) => p.name.toLowerCase() === normalized);
  return prov ? prov.cities.map((c) => c.name) : [];
}

export function getDistrictsByCity(
  provinceName: string,
  cityName: string,
): Array<{ name: string; postalCode: string }> {
  if (!provinceName || !cityName) return [];
  const normalizedProv = provinceName.trim().toLowerCase();
  const normalizedCity = cityName.trim().toLowerCase();
  const prov = INDONESIA_REGIONS.find((p) => p.name.toLowerCase() === normalizedProv);
  if (!prov) return [];
  const city = prov.cities.find(
    (c) =>
      c.name.toLowerCase() === normalizedCity ||
      c.name.toLowerCase().includes(normalizedCity) ||
      normalizedCity.includes(c.name.toLowerCase()),
  );
  if (!city) return [];
  return city.districts.map((d) => ({ name: d.name, postalCode: d.postalCode }));
}

export function findPostalCode(
  provinceName: string,
  cityName: string,
  districtName: string,
): string | null {
  if (!provinceName || !cityName || !districtName) return null;
  const districts = getDistrictsByCity(provinceName, cityName);
  const normalizedDistrict = districtName.trim().toLowerCase();
  const found = districts.find(
    (d) =>
      d.name.toLowerCase() === normalizedDistrict ||
      d.name.toLowerCase().includes(normalizedDistrict) ||
      normalizedDistrict.includes(d.name.toLowerCase()),
  );
  return found ? found.postalCode : null;
}
