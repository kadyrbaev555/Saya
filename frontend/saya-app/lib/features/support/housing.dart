import 'package:flutter_map/flutter_map.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:geolocator/geolocator.dart';
import 'package:latlong2/latlong.dart';
import 'package:material_ui/material_ui.dart';

enum RentalArea {
  city('Весь Алматы', 'almaty', LatLng(43.2383, 76.9455)),
  almaly('Алмалинский район', 'almaty-almalinskij', LatLng(43.251, 76.918)),
  bostandyk(
    'Бостандыкский район',
    'almaty-bostandykskij',
    LatLng(43.209, 76.898),
  ),
  medeu('Медеуский район', 'almaty-medeuskij', LatLng(43.239, 76.967));

  const RentalArea(this.label, this.slug, this.center);
  final String label, slug;
  final LatLng center;
  Uri get listings =>
      Uri.https('krisha.kz', '/map/arenda/kvartiry-posutochno/$slug/');

  static RentalArea near(LatLng location) {
    var nearest = city;
    var limit = 8000.0;
    for (final area in values.skip(1)) {
      final distance = Geolocator.distanceBetween(
        location.latitude,
        location.longitude,
        area.center.latitude,
        area.center.longitude,
      );
      if (distance < limit) {
        nearest = area;
        limit = distance;
      }
    }
    return nearest;
  }
}

final locateProvider = Provider<Future<LatLng?> Function()>(
  (ref) => () async {
    if (!await Geolocator.isLocationServiceEnabled()) return null;
    var permission = await Geolocator.checkPermission();
    if (permission == LocationPermission.denied)
      permission = await Geolocator.requestPermission();
    if (permission != LocationPermission.always &&
        permission != LocationPermission.whileInUse)
      return null;
    final p = await Geolocator.getCurrentPosition(
      locationSettings: const LocationSettings(
        accuracy: LocationAccuracy.low,
        timeLimit: Duration(seconds: 8),
      ),
    );
    return LatLng(p.latitude, p.longitude);
  },
);

class HousingSearch extends ConsumerStatefulWidget {
  const HousingSearch({super.key, required this.open});
  final Future<void> Function(Uri) open;
  @override
  ConsumerState<HousingSearch> createState() => _HousingSearchState();
}

class _HousingSearchState extends ConsumerState<HousingSearch> {
  RentalArea area = RentalArea.city;
  bool map = false, busy = false, tilesUnavailable = false;
  String? notice;
  Future<void> locate() async {
    setState(() {
      busy = true;
      notice = null;
    });
    LatLng? point;
    try {
      point = await ref.read(locateProvider)();
    } catch (_) {
      point = null;
    }
    if (!mounted) return;
    setState(() {
      busy = false;
      if (point == null) {
        notice = 'Местоположение недоступно. Выберите район вручную.';
        return;
      }
      area = RentalArea.near(point);
      notice =
          area == RentalArea.city
              ? 'Поблизости нет районов из списка. Можно выбрать район вручную.'
              : 'Выбран ближайший из ориентиров. При необходимости измените район.';
    });
  }

  @override
  Widget build(BuildContext context) => Column(
    crossAxisAlignment: CrossAxisAlignment.stretch,
    children: [
      Text(
        'Посуточная аренда в Алматы',
        style: Theme.of(context).textTheme.titleLarge,
      ),
      const SizedBox(height: 16),
      DropdownButton<RentalArea>(
        value: area,
        isExpanded: true,
        items: [
          for (final a in RentalArea.values)
            DropdownMenuItem(value: a, child: Text(a.label)),
        ],
        onChanged:
            busy
                ? null
                : (v) => setState(() {
                  area = v!;
                  notice = null;
                }),
      ),
      TextButton.icon(
        onPressed: busy ? null : locate,
        icon: const Icon(Icons.my_location),
        label: Text(busy ? 'Определяем район…' : 'Подобрать район рядом'),
      ),
      if (notice != null) Text(notice!),
      if (tilesUnavailable)
        const Text(
          'Карта не загрузилась. Можно выбрать район из списка или открыть Krisha.kz.',
        ),
      if (!map)
        OutlinedButton(
          onPressed:
              () => setState(() {
                map = true;
                tilesUnavailable = false;
              }),
          child: const Text('Показать мини-карту'),
        )
      else
        SizedBox(
          height: 230,
          child: FlutterMap(
            key: ValueKey(area),
            options: MapOptions(
              initialCenter: area.center,
              initialZoom: area == RentalArea.city ? 11 : 13,
            ),
            children: [
              TileLayer(
                urlTemplate: 'https://tile.openstreetmap.org/{z}/{x}/{y}.png',
                userAgentPackageName: 'SAYA/1.3 (github.com/kadyrbaev555/Saya)',
                panBuffer: 0,
                errorTileCallback: (_, __, ___) {
                  if (mounted && map)
                    setState(() {
                      map = false;
                      tilesUnavailable = true;
                    });
                },
              ),
              MarkerLayer(
                markers: [
                  for (final a in RentalArea.values.skip(1))
                    Marker(
                      point: a.center,
                      width: 44,
                      height: 44,
                      child: IconButton(
                        tooltip: a.label,
                        onPressed: busy ? null : () => setState(() => area = a),
                        icon: Icon(
                          Icons.location_on,
                          color: a == area ? Colors.deepOrange : Colors.teal,
                        ),
                      ),
                    ),
                ],
              ),
              Align(
                alignment: Alignment.bottomRight,
                child: ColoredBox(
                  color: Colors.white,
                  child: InkWell(
                    onTap:
                        () => widget.open(
                          Uri.parse('https://www.openstreetmap.org/copyright'),
                        ),
                    child: const Text(
                      '© OpenStreetMap contributors',
                      style: TextStyle(color: Colors.black),
                    ),
                  ),
                ),
              ),
            ],
          ),
        ),
      const SizedBox(height: 12),
      const Text(
        'На мини-карте отмечены районы. Объявления и цены доступны на Krisha.kz.',
      ),
      FilledButton(
        onPressed: () => widget.open(area.listings),
        child: const Text('Посмотреть квартиры на Krisha.kz'),
      ),
    ],
  );
}
