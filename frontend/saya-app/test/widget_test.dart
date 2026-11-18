import 'package:material_ui/material_ui.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:saya_app/app/app.dart';
import 'package:saya_app/core/network/api.dart';
import 'package:saya_app/features/auth/session.dart';
import 'package:saya_app/core/realtime/realtime.dart';

class FakeApi extends Api {
  bool fail = false;
  bool submitted = false;
  @override
  Future<String> startRegistration(
    String name,
    String phone,
    String email,
    String password,
  ) async {
    return 'test-challenge';
  }

  @override
  Future<Map<String, dynamic>> confirmRegistration(
    String id,
    String code,
  ) async {
    if (id != 'test-challenge' || code != '123456') throw ApiError(401);
    return {'id': 'demo', 'email': 'demo@example.test', 'role': 'USER'};
  }

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
    if (path == '/surveys/latest')
      return {'id': submitted ? 'demo-assessment' : ''};
    if (path == '/surveys/interview/focus') {
      expect(body?['consent'], isTrue);
      expect(body!.keys, unorderedEquals(['topic', 'ratings', 'consent']));
      return {'focus': 'resources'};
    }
    if (path == '/surveys/interview')
      return {
        'id': 'saya-week',
        'version': 2,
        'disclaimer': 'Демонстрационный опрос',
        'questions': [
          {
            'id': 'housing',
            'label': 'Где вы жили?',
            'variants': {'resources': 'Что помогало с жильём?'},
          },
          {'id': 'priority', 'label': 'Демонстрация: информация'},
        ],
      };
    if (path == '/surveys/submissions') {
      submitted = true;
      expect(body?['ratings'], {'housing': 1});
      expect(body?['responses'], {'housing': 'Ищу жильё'});
      expect(body?['external_llm_consent'], false);
      return {'id': 'demo-assessment', 'status': 'pending'};
    }
    if (path.startsWith('/assessments/'))
      return {
        'id': 'demo-assessment',
        'status': 'completed',
        'provider': 'self-report',
        'version': 2,
        'ratings': {'housing': 1},
        'directions': ['housing'],
        'summary': 'Подтверждено',
        'disclaimer': 'Не диагноз',
        'recommendations': ['Демонстрационная рекомендация'],
      };
    return {'id': 'demo', 'email': 'demo@example.test', 'role': 'USER'};
  }

  @override
  Future<void> logout() async {}
}

void main() {
  testWidgets(
    'Guest catalog returns to the originating form and emergency contacts open without login',
    (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [apiProvider.overrideWithValue(FakeApi())],
          child: const SayaApp(),
        ),
      );
      for (final label in ['Войти', 'Создать аккаунт']) {
        await tester.tap(find.text(label).last);
        await tester.pumpAndSettle();
        await tester.ensureVisible(find.text('Помочь человеку'));
        await tester.tap(find.text('Помочь человеку'));
        await tester.pumpAndSettle();
        expect(
          find.text('Пока нет опубликованных анкет для помощи.'),
          findsOneWidget,
        );
        await tester.pageBack();
        await tester.pumpAndSettle();
        expect(find.text(label), findsWidgets);
      }
    },
  );

  testWidgets('Emergency directory is accessible from welcome', (tester) async {
    await tester.pumpWidget(const ProviderScope(child: SayaApp()));
    await tester.tap(find.text('Экстренная помощь'));
    await tester.pumpAndSettle();
    expect(find.text('112'), findsOneWidget);
    expect(find.text('103'), findsOneWidget);
    expect(find.text('104'), findsOneWidget);
  });
  testWidgets('Registration requests a code before opening the home screen', (
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
    await tester.tap(find.text('Создать аккаунт'));
    await tester.pumpAndSettle();
    final fields = find.byType(TextFormField);
    for (final (index, value)
        in [
          'Test User',
          '+77000000001',
          'demo@example.test',
          'demo-password-long',
        ].indexed) {
      await tester.enterText(fields.at(index), value);
    }
    await tester.tap(find.text('Получить код'));
    await tester.pumpAndSettle();
    expect(find.text('Подтверждение почты'), findsOneWidget);
    await tester.enterText(fields, '000001');
    await tester.tap(find.text('Подтвердить и войти'));
    await tester.pumpAndSettle();
    expect(find.textContaining('Код неверен'), findsOneWidget);
    await tester.enterText(fields, '123456');
    await tester.tap(find.text('Подтвердить и войти'));
    await tester.pumpAndSettle();
    expect(find.text('Какая помощь вам нужна?'), findsOneWidget);
    expect(find.text('Денежная поддержка'), findsNothing);
    for (final label in [
      'Жильё',
      'Поиск работы',
      'Психолог',
      'Социальная поддержка',
      'Экстренная помощь',
    ]) {
      await tester.ensureVisible(find.text(label));
      await tester.tap(find.text(label));
      await tester.pumpAndSettle();
      expect(find.text(label), findsOneWidget);
      await tester.pageBack();
      await tester.pumpAndSettle();
    }
  });
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
    await tester.tap(find.text('Профиль'));
    await tester.pumpAndSettle();
    expect(find.text('demo@example.test'), findsOneWidget);
    await tester.tap(find.text('Пройти опрос о неделе'));
    await tester.pumpAndSettle();
    await tester.tap(find.byType(SwitchListTile));
    await tester.pumpAndSettle();
    expect(find.text('Что помогало с жильём?'), findsOneWidget);
    await tester.enterText(find.byType(TextFormField), 'Ищу жильё');
    await tester.tap(find.text('1'));
    await tester.ensureVisible(find.text('Дальше'));
    await tester.tap(find.text('Дальше'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Пропустить вопрос'));
    await tester.pumpAndSettle();
    final consent = find.text(
      'Подтверждаю описание и оценки. Сохранить их и ответы в SAYA.',
    );
    await tester.scrollUntilVisible(
      consent,
      240,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.tap(consent);
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Сохранить результат'));
    await tester.tap(find.text('Сохранить результат'));
    await tester.pumpAndSettle();
    expect(find.text('Результат готов'), findsOneWidget);
    expect(find.text('Подтверждено'), findsOneWidget);
    await tester.tap(find.text('Вернуться в профиль'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Последний результат'));
    await tester.pumpAndSettle();
    expect(find.text('Подтверждено'), findsOneWidget);
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
