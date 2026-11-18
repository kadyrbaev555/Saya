import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:latlong2/latlong.dart';
import 'package:material_ui/material_ui.dart';
import 'package:saya_app/features/support/housing.dart';
import 'package:saya_app/features/support/jobs.dart';
import 'package:saya_app/features/support/support.dart';
import 'package:saya_app/features/auth/session.dart';
import 'package:saya_app/core/network/api.dart';

class JobsApi extends Api {
  bool down = false;
  List<Map<String, dynamic>> items = [];
  @override
  Future<Map<String, dynamic>> request(
    String path, {
    Map<String, dynamic>? body,
  }) async {
    expect(path, '/support/vacancies');
    expect(body, {'text': 'Go', 'area': '160'});
    if (down) throw ApiError(503);
    return {'items': items};
  }
}

class ProfileApi extends Api {
  @override
  Future<Map<String, dynamic>> request(
    String path, {
    Map<String, dynamic>? body,
  }) async {
    if (path == '/surveys/latest') return {'id': 'one'};
    return {'status': 'completed', 'summary': 'Черновик обращения'};
  }
}

void main() {
  for (final item in [
    (Direction.psychologist, 'Набрать 1303', 'tel:1303'),
    (
      Direction.social,
      'Написать Birgemiz в Telegram',
      'https://t.me/BIRGEMIZ2020',
    ),
    (Direction.emergency, 'Скорая медицинская помощь', 'tel:103'),
  ]) {
    testWidgets(
      'Contact action ${item.$1.name} opens only the selected destination',
      (tester) async {
        Uri? opened;
        final api = ProfileApi();
        addTearDown(api.close);
        await tester.pumpWidget(
          ProviderScope(
            overrides: [
              apiProvider.overrideWithValue(api),
              externalLinkProvider.overrideWithValue((uri) async {
                opened = uri;
                return true;
              }),
            ],
            child: MaterialApp(home: DirectionPage(item.$1)),
          ),
        );
        await tester.pumpAndSettle();
        if (item.$1 != Direction.emergency) {
          expect(find.text('Черновик обращения'), findsOneWidget);
          await tester.enterText(find.byType(TextField), 'Личное обращение');
        }
        expect(opened, isNull);
        await tester.ensureVisible(find.text(item.$2));
        await tester.tap(find.text(item.$2));
        expect(opened.toString(), item.$3);
      },
    );
  }
  test('A distant position leaves city selection to the user', () {
    expect(RentalArea.near(const LatLng(51.17, 71.43)), RentalArea.city);
    expect(RentalArea.near(RentalArea.bostandyk.center), RentalArea.bostandyk);
  });
  testWidgets('Location denial preserves manual rental search', (tester) async {
    Uri? opened;
    await tester.pumpWidget(
      ProviderScope(
        overrides: [locateProvider.overrideWithValue(() async => null)],
        child: MaterialApp(
          home: Scaffold(
            body: HousingSearch(
              open: (url) async {
                opened = url;
              },
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text('Подобрать район рядом'));
    await tester.pumpAndSettle();
    expect(find.textContaining('вручную'), findsOneWidget);
    await tester.tap(find.byType(DropdownButton<RentalArea>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Алмалинский район').last);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Посмотреть квартиры на Krisha.kz'));
    expect(opened, RentalArea.almaly.listings);
  });
  testWidgets('Vacancies handle empty, rejected and successful searches', (
    tester,
  ) async {
    final api = JobsApi();
    addTearDown(api.close);
    Uri? opened;
    await tester.pumpWidget(
      ProviderScope(
        overrides: [apiProvider.overrideWithValue(api)],
        child: MaterialApp(
          home: Scaffold(
            body: JobSearch(
              open: (url) async {
                opened = url;
              },
            ),
          ),
        ),
      ),
    );
    await tester.enterText(find.byType(TextField), 'Go');
    await tester.pump();
    Future<void> search() async {
      await tester.tap(find.text('Искать на HeadHunter'));
      await tester.pumpAndSettle();
    }

    await search();
    expect(find.textContaining('не найдено'), findsOneWidget);
    api.down = true;
    await search();
    expect(find.textContaining('недоступен'), findsOneWidget);
    await tester.tap(find.text('Открыть этот поиск на hh.kz'));
    expect(opened?.queryParameters, {'area': '160', 'text': 'Go'});
    api.down = false;
    api.items = [
      {
        'title': 'Go developer',
        'employer': 'Example',
        'url': 'https://hh.kz/vacancy/12',
      },
    ];
    await search();
    await tester.tap(find.text('Go developer'));
    expect(opened.toString(), 'https://hh.kz/vacancy/12');
  });
}
