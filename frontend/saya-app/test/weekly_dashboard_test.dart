import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:saya_app/features/assessments/weekly.dart';

void main() {
  testWidgets(
    'Dashboard compares confirmed ratings and recommendations across two weeks',
    (tester) async {
      final history = {
        'entries': [
          {
            'id': 'new',
            'completed_at': 1700604800,
            'version': 2,
            'percentage': 60,
            'ratings': {'housing': 3},
            'directions': ['jobs'],
          },
          {
            'id': 'old',
            'completed_at': 1700000000,
            'version': 2,
            'percentage': 50,
            'ratings': {'housing': 1},
            'directions': ['housing'],
          },
        ],
        'reminder_due': true,
      };
      await tester.pumpWidget(
        ProviderScope(
          overrides: [weeklyProvider('').overrideWith((ref) async => history)],
          child: const MaterialApp(home: WeeklyDashboard()),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('Пора пройти опрос о неделе'), findsOneWidget);
      expect(find.text('Общая оценка выросла на 10 п. п.'), findsOneWidget);
      expect(find.text('Жильё · 3/4 (+2)'), findsOneWidget);
      expect(find.text('Добавились · Поиск работы'), findsOneWidget);
      expect(find.text('Больше не предложены · Жильё'), findsOneWidget);
    },
  );
}
