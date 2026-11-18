import 'dart:async';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';
import '../../core/realtime/realtime.dart';
import '../auth/session.dart';
import '../surveys/survey.dart';
import 'android_reminders.dart';

final weeklyProvider = FutureProvider.autoDispose.family<
  Map<String, dynamic>,
  String
>((ref, cursor) async {
  final owner = ref.watch(sessionProvider).asData?.value?['id'];
  if (owner == null) return {};
  ref.watch(realtimeProvider);
  final api = ref.watch(apiProvider);
  final data = await api.request(
    '/weekly/history${cursor.isEmpty ? '' : '?before=${Uri.encodeQueryComponent(cursor)}'}',
  );
  final due = data['next_due'] as int? ?? 0;
  if (ref.mounted &&
      cursor.isEmpty &&
      due > 0 &&
      data['reminder_due'] != true) {
    final remaining = DateTime.fromMillisecondsSinceEpoch(
      due * 1000,
    ).difference(DateTime.now());
    final timer = Timer(
      remaining.isNegative
          ? const Duration(minutes: 1)
          : remaining + const Duration(seconds: 1),
      () => ref.invalidateSelf(),
    );
    ref.onDispose(timer.cancel);
  }
  return data;
});

class WeeklyInvite extends ConsumerWidget {
  const WeeklyInvite({super.key});
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final due =
        ref.watch(weeklyProvider('')).asData?.value['reminder_due'] == true;
    if (!due) return const SizedBox.shrink();
    return Card(
      child: ListTile(
        leading: const Icon(Icons.event_repeat),
        title: const Text('Пора пройти опрос о неделе'),
        trailing: const Icon(Icons.chevron_right),
        onTap: () => context.push('/survey'),
      ),
    );
  }
}

class WeeklyPushButton extends ConsumerStatefulWidget {
  const WeeklyPushButton({super.key});
  @override
  ConsumerState<WeeklyPushButton> createState() => _WeeklyPushButtonState();
}

class _WeeklyPushButtonState extends ConsumerState<WeeklyPushButton> {
  bool busy = false;
  @override
  Widget build(BuildContext context) {
    if (!AndroidReminders.supported) return const SizedBox.shrink();
    final history = ref.watch(weeklyProvider('')).asData?.value;
    final owner = ref.watch(sessionProvider).asData?.value?['id'] as String?;
    if (owner == null || history == null || history['push_available'] != true)
      return const SizedBox.shrink();
    final push = AndroidReminders.instance;
    return ValueListenableBuilder<String?>(
      valueListenable: push.enabledFor,
      builder:
          (context, active, _) => OutlinedButton.icon(
            icon: const Icon(Icons.notifications_outlined),
            label: Text(
              active == owner
                  ? 'Отключить push на этом телефоне'
                  : 'Напомнить на этом телефоне через неделю',
            ),
            onPressed:
                busy
                    ? null
                    : () async {
                      setState(() => busy = true);
                      try {
                        final api = ref.read(apiProvider);
                        if (active == owner) {
                          await push.disable(api);
                        } else {
                          await push.enable(api, owner);
                        }
                      } catch (_) {
                        if (context.mounted)
                          ScaffoldMessenger.of(context).showSnackBar(
                            const SnackBar(
                              content: Text(
                                'Не удалось изменить напоминания. Проверьте разрешение на уведомления и подключение.',
                              ),
                            ),
                          );
                      } finally {
                        if (mounted) setState(() => busy = false);
                      }
                    },
          ),
    );
  }
}

String surveyDate(int seconds) {
  final d = DateTime.fromMillisecondsSinceEpoch(seconds * 1000);
  String two(int n) => n.toString().padLeft(2, '0');
  return '${two(d.day)}.${two(d.month)}.${d.year}, ${two(d.hour)}:${two(d.minute)}';
}

String delta(int n) => n > 0 ? '+$n' : '$n';

class WeeklyDashboard extends ConsumerStatefulWidget {
  const WeeklyDashboard({super.key});
  @override
  ConsumerState<WeeklyDashboard> createState() => _WeeklyDashboardState();
}

class _WeeklyDashboardState extends ConsumerState<WeeklyDashboard> {
  final pages = [''];
  @override
  Widget build(BuildContext context) {
    final cursor = pages.last;
    return Scaffold(
      appBar: AppBar(title: const Text('Динамика по неделям')),
      body: SafeArea(
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 680),
            child: ref
                .watch(weeklyProvider(cursor))
                .when(
                  loading:
                      () => const Center(child: CircularProgressIndicator()),
                  error:
                      (_, __) => TextButton(
                        onPressed: () => ref.invalidate(weeklyProvider(cursor)),
                        child: const Text('Повторить загрузку истории'),
                      ),
                  data: (data) {
                    final list = data['entries'] as List? ?? [];
                    final next = data['next_cursor'] as String? ?? '';
                    return ListView(
                      padding: const EdgeInsets.all(16),
                      children: [
                        if (list.isEmpty)
                          const Text(
                            'Здесь появятся результаты завершённых опросов.',
                          ),
                        if (data['reminder_due'] == true)
                          const WeeklyInvite()
                        else if ((data['next_due'] ?? 0) > 0)
                          Text(
                            'Следующий опрос · ${surveyDate(data['next_due'])}',
                          ),
                        if (list.isNotEmpty) const WeeklyPushButton(),
                        for (var i = 0; i < list.length; i++)
                          _WeekCard(
                            entry: list[i],
                            previous: i + 1 < list.length ? list[i + 1] : null,
                          ),
                        if (next.isNotEmpty)
                          TextButton(
                            onPressed: () => setState(() => pages.add(next)),
                            child: const Text('Более ранние опросы'),
                          ),
                        if (pages.length > 1)
                          TextButton(
                            onPressed: () => setState(() => pages.removeLast()),
                            child: const Text('Более новые опросы'),
                          ),
                      ],
                    );
                  },
                ),
          ),
        ),
      ),
    );
  }
}

class _WeekCard extends StatelessWidget {
  const _WeekCard({required this.entry, this.previous});
  final Map entry;
  final Map? previous;
  @override
  Widget build(BuildContext context) {
    final percentage = entry['percentage'] as int?;
    final comparable = entry['version'] == 2 && previous?['version'] == 2;
    final old = comparable ? (previous?['percentage'] as int?) : null;
    final ratings = entry['ratings'] as Map? ?? {};
    final oldRatings = comparable ? ((previous?['ratings'] as Map?) ?? {}) : {};
    final directions =
        (entry['directions'] as List? ?? []).whereType<String>().toSet();
    final prior =
        (previous?['directions'] as List? ?? []).whereType<String>().toSet();
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              surveyDate(entry['completed_at']),
              style: Theme.of(context).textTheme.titleMedium,
            ),
            if (percentage != null) ...[
              Text('Самооценка · $percentage%'),
              LinearProgressIndicator(value: percentage / 100),
            ],
            if (old != null && percentage != null)
              Text(
                percentage == old
                    ? 'Без изменения общей оценки'
                    : '${percentage > old ? 'Общая оценка выросла' : 'Общая оценка снизилась'} на ${(percentage - old).abs()} п. п.',
              ),
            for (final e in ratings.entries)
              Text(
                '${topicLabels[e.key] ?? e.key} · ${e.value}/4${oldRatings[e.key] == null ? '' : ' (${delta(e.value - oldRatings[e.key])})'}',
              ),
            if (directions.isNotEmpty)
              Text(
                'Рекомендации · ${directions.map((k) => needLabels[k] ?? k).join(', ')}',
              ),
            if (comparable && directions.difference(prior).isNotEmpty)
              Text(
                'Добавились · ${directions.difference(prior).map((k) => needLabels[k] ?? k).join(', ')}',
              ),
            if (comparable && prior.difference(directions).isNotEmpty)
              Text(
                'Больше не предложены · ${prior.difference(directions).map((k) => needLabels[k] ?? k).join(', ')}',
              ),
            TextButton(
              onPressed: () => context.push('/assessment/${entry['id']}'),
              child: const Text('Открыть результат'),
            ),
          ],
        ),
      ),
    );
  }
}
