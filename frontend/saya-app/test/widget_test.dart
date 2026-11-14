import 'package:material_ui/material_ui.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:saya_app/app/app.dart';
import 'package:saya_app/core/network/api.dart';
import 'package:saya_app/features/auth/session.dart';
import 'package:saya_app/core/realtime/realtime.dart';

class FakeApi extends Api {
  bool fail = false;
  @override
  Future<void> authenticate(
    String action,
    String email,
    String password,
  ) async {
    if (fail) throw ApiError(401);
  }

  @override
  Future<Map<String, dynamic>> request(
    String path, {
    Map<String, dynamic>? body,
  }) async {
    if (path == '/surveys/current')
      return {
        'id': 'saya-demo',
        'version': 1,
        'disclaimer': 'Демонстрационный опрос',
        'questions': [
          {'id': 'immediate_threat', 'label': 'Демонстрация: угроза'},
          {'id': 'information_requested', 'label': 'Демонстрация: информация'},
        ],
      };
    if (path == '/surveys/submissions')
      return {'id': 'demo-assessment', 'status': 'pending'};
    if (path.startsWith('/assessments/'))
      return {
        'id': 'demo-assessment',
        'status': 'completed',
        'provider': 'demonstration-mock',
        'disclaimer': 'Не диагноз',
        'recommendations': ['Демонстрационная рекомендация'],
      };
    return {'id': 'demo', 'email': 'demo@example.test', 'role': 'USER'};
  }

  @override
  Future<void> logout() async {}
}

void main() {
  testWidgets('Welcome opens login form', (tester) async {
    await tester.pumpWidget(const ProviderScope(child: SayaApp()));
    expect(find.text('Поддержка в вашем темпе'), findsOneWidget);
    await tester.tap(find.text('Войти'));
    await tester.pumpAndSettle();
    expect(find.byType(TextField), findsNWidgets(2));
  });
  testWidgets('Login, consent, survey submission and result navigation', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          apiProvider.overrideWithValue(FakeApi()),
          realtimeProvider.overrideWith((ref) => const Stream.empty()),
        ],
        child: const SayaApp(),
      ),
    );
    await tester.tap(find.text('Войти'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField).first, 'demo@example.test');
    await tester.enterText(
      find.byType(TextField).last,
      'demonstration-password',
    );
    await tester.tap(find.text('Продолжить'));
    await tester.pumpAndSettle();
    expect(find.text('demo@example.test'), findsOneWidget);
    await tester.tap(find.text('Открыть демонстрационный опрос'));
    await tester.pumpAndSettle();
    await tester.tap(find.byType(CheckboxListTile).first);
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Отправить демонстрационные ответы'));
    await tester.tap(find.text('Отправить демонстрационные ответы'));
    await tester.pumpAndSettle();
    expect(find.text('Результат готов'), findsOneWidget);
    expect(find.text('Демонстрационная рекомендация'), findsOneWidget);
  });
  test('Session reports success, failure and logout', () async {
    final fake = FakeApi();
    final container = ProviderContainer(
      overrides: [apiProvider.overrideWithValue(fake)],
    );
    addTearDown(container.dispose);
    final notifier = container.read(sessionProvider.notifier);
    await notifier.authenticate('login', 'demo@example.test', 'password');
    expect(container.read(sessionProvider).asData!.value!['role'], 'USER');
    await notifier.logout();
    expect(container.read(sessionProvider).asData!.value, isNull);
    fake.fail = true;
    await notifier.authenticate('login', 'demo@example.test', 'wrong');
    expect(container.read(sessionProvider).hasError, isTrue);
  });
}
