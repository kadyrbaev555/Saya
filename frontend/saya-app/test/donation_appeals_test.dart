import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:saya_app/features/support/appeals.dart';

void main() {
  testWidgets(
    'Public listing shows payment actions only for a connected recipient',
    (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            appealsProvider('public').overrideWith(
              (ref) async => {
                'checkout_available': true,
                'items': [
                  {
                    'id': 'a',
                    'description': 'Помощь с продуктами',
                    'card_ready': false,
                  },
                  {
                    'id': 'b',
                    'description': 'Помощь с жильём',
                    'card_ready': true,
                  },
                ],
              },
            ),
          ],
          child: const MaterialApp(home: AppealsPage('public')),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('Тестовый платёж · 500 ₸'), findsOneWidget);
      expect(
        find.text('Тестовый платёж для этой анкеты пока недоступен.'),
        findsOneWidget,
      );
      expect(find.text('Пожаловаться'), findsNWidgets(2));
      expect(find.text('Одобрить'), findsNothing);
    },
  );
  testWidgets('Editing the public text resets consent', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          home: Scaffold(
            body: Consumer(
              builder:
                  (context, ref, _) => TextButton(
                    onPressed:
                        () => editAppeal(context, ref, 'id', 'Моё описание'),
                    child: const Text('Анкета'),
                  ),
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text('Анкета'));
    await tester.pumpAndSettle();
    final send = find.widgetWithText(FilledButton, 'Отправить на проверку');
    expect(tester.widget<FilledButton>(send).onPressed, isNull);
    await tester.tap(find.byType(Checkbox));
    await tester.pumpAndSettle();
    expect(tester.widget<FilledButton>(send).onPressed, isNotNull);
    await tester.enterText(find.byType(TextField), 'Описание для публикации');
    await tester.pumpAndSettle();
    expect(tester.widget<FilledButton>(send).onPressed, isNull);
  });
}
