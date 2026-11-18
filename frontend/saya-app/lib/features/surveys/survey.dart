import 'dart:convert';
import 'dart:math';
import 'package:material_ui/material_ui.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../auth/session.dart';

final surveyProvider = FutureProvider(
  (ref) => ref.read(apiProvider).request('/surveys/interview'),
);
String operationKey() {
  final random = Random.secure();
  final b = List<int>.generate(16, (_) => random.nextInt(256));
  b[6] = (b[6] & 15) | 64;
  b[8] = (b[8] & 63) | 128;
  final s = b.map((v) => v.toRadixString(16).padLeft(2, '0')).join();
  return '${s.substring(0, 8)}-${s.substring(8, 12)}-${s.substring(12, 16)}-${s.substring(16, 20)}-${s.substring(20)}';
}

class SurveyPage extends ConsumerStatefulWidget {
  const SurveyPage({super.key});
  @override
  ConsumerState<SurveyPage> createState() => _SurveyPageState();
}

const topicLabels = {
  'housing': 'Жильё',
  'expenses': 'Базовые расходы',
  'work': 'Работа',
  'connection': 'Общение',
  'daily': 'Повседневные дела',
  'priority': 'Приоритет',
};
const needLabels = {
  'housing': 'Жильё',
  'jobs': 'Поиск работы',
  'psychologist': 'Психолог',
  'social': 'Социальная поддержка',
  'financial': 'Денежная поддержка',
  'emergency': 'Экстренная помощь',
};
const ratingLabels = [
  'Совсем не отвечала',
  'В малой степени',
  'Частично',
  'В основном',
  'Полностью',
];

class _SurveyPageState extends ConsumerState<SurveyPage> {
  final responses = <String, String>{};
  final ratings = <String, int>{};
  final needs = <String>{};
  final focusByTopic = <String, String>{};
  bool assisted = false, choosing = false;
  int focusRequest = 0;
  String? focusNotice;

  Future<void> selectQuestion(String id) async {
    final ticket = ++focusRequest;
    setState(() {
      choosing = true;
      focusNotice = null;
    });
    try {
      final result = await ref
          .read(apiProvider)
          .request(
            '/surveys/interview/focus',
            body: {'topic': id, 'ratings': Map.of(ratings), 'consent': true},
          );
      if (mounted && ticket == focusRequest)
        setState(() => focusByTopic[id] = result['focus'] as String);
    } catch (_) {
      if (mounted && ticket == focusRequest)
        setState(() {
          focusNotice =
              'ИИ сейчас недоступен. Можно продолжить с основным вопросом.';
          assisted = false;
        });
    } finally {
      if (mounted && ticket == focusRequest) setState(() => choosing = false);
    }
  }

  final summary = TextEditingController();
  String requestKey = operationKey();
  String? previousPayload, error, lastDraft;
  bool consent = false, external = false, urgent = false, loading = false;
  int step = 0;
  @override
  void dispose() {
    summary.dispose();
    super.dispose();
  }

  void review() {
    final draft = responses.entries
        .where((e) => e.value.trim().isNotEmpty)
        .map((e) => '${topicLabels[e.key]} — ${e.value.trim()}')
        .join('\n');
    if (lastDraft == null || summary.text == lastDraft) {
      summary.text = String.fromCharCodes(draft.runes.take(1800));
    }
    lastDraft = String.fromCharCodes(draft.runes.take(1800));
    if ((ratings['expenses'] ?? 4) > 1) needs.remove('financial');
  }

  Future<void> submit(Map<String, dynamic> survey) async {
    final body = <String, dynamic>{
      'questionnaire_id': survey['id'],
      'version': survey['version'],
      'responses': {
        for (final e in responses.entries)
          if (e.value.trim().isNotEmpty) e.key: e.value.trim(),
      },
      'ratings': ratings,
      'question_focus': focusByTopic,
      'summary': summary.text.trim(),
      'needs': needs.toList(),
      'answers': {'immediate_threat': urgent},
      'consent': consent,
      'external_llm_consent': external,
    };
    final payload = jsonEncode(body);
    if (previousPayload != payload) {
      requestKey = operationKey();
      previousPayload = payload;
    }
    setState(() {
      loading = true;
      error = null;
    });
    try {
      final result = await ref
          .read(apiProvider)
          .request(
            '/surveys/submissions',
            body: {...body, 'idempotency_key': requestKey},
          );
      if (mounted) context.go('/assessment/${result['id']}');
    } catch (_) {
      if (mounted)
        setState(
          () => error = 'Не удалось сохранить ответы. Попробуйте ещё раз.',
        );
    } finally {
      if (mounted) setState(() => loading = false);
    }
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(title: const Text('Моя неделя')),
    body: Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 640),
        child: ref
            .watch(surveyProvider)
            .when(
              loading: () => const CircularProgressIndicator(),
              error:
                  (_, __) => TextButton(
                    onPressed: () => ref.invalidate(surveyProvider),
                    child: const Text('Загрузить вопросы повторно'),
                  ),
              data: (data) {
                final qs = data['questions'] as List;
                final reviewing = step == qs.length;
                final id = reviewing ? '' : qs[step]['id'] as String;
                void next() {
                  setState(() {
                    step++;
                    if (step == qs.length) review();
                  });
                  if (step < qs.length &&
                      assisted &&
                      !focusByTopic.containsKey(qs[step]["id"]))
                    selectQuestion(qs[step]["id"] as String);
                }

                return ListView(
                  key: ValueKey(step),
                  padding: const EdgeInsets.all(24),
                  children: [
                    LinearProgressIndicator(
                      value: (step + 1) / (qs.length + 1),
                    ),
                    const SizedBox(height: 16),
                    if (!reviewing) ...[
                      SwitchListTile(
                        title: const Text('Подбирать вопросы с Gemini'),
                        subtitle: const Text(
                          'Разрешаю передавать Gemini тему и мои оценки для подбора следующего вопроса.',
                        ),
                        value: assisted,
                        onChanged: (value) {
                          setState(() {
                            assisted = value;
                            focusNotice = null;
                            if (!value) {
                              focusRequest++;
                              choosing = false;
                            }
                          });
                          if (value && (responses[id] ?? '').isEmpty)
                            selectQuestion(id);
                        },
                      ),
                      if (choosing) const LinearProgressIndicator(),
                      if (focusNotice != null) Text(focusNotice!),

                      Text(
                        'Вопрос ${step + 1} из ${qs.length} · ${topicLabels[id]}',
                      ),
                      if (step == 0)
                        Padding(
                          padding: const EdgeInsets.symmetric(vertical: 12),
                          child: Text(data['disclaimer'] as String),
                        ),
                      Text(
                        (qs[step]['variants'] as Map?)?[focusByTopic[id]]
                                as String? ??
                            qs[step]['label'] as String,
                        style: Theme.of(context).textTheme.titleLarge,
                      ),
                      const SizedBox(height: 12),
                      TextFormField(
                        enabled: !choosing,
                        initialValue: responses[id],
                        minLines: 2,
                        maxLines: 5,
                        maxLength: 800,
                        decoration: const InputDecoration(
                          labelText: 'Ваш ответ',
                        ),
                        onChanged: (s) => responses[id] = s,
                      ),
                      if (id != 'priority') ...[
                        const Text(
                          'Насколько ситуация в этой сфере отвечала вашим потребностям?',
                        ),
                        Wrap(
                          spacing: 6,
                          children: [
                            for (var i = 0; i < 5; i++)
                              ChoiceChip(
                                label: Text('$i'),
                                selected: ratings[id] == i,
                                onSelected:
                                    (v) => setState(() {
                                      if (v) {
                                        ratings[id] = i;
                                      } else {
                                        ratings.remove(id);
                                      }
                                    }),
                              ),
                          ],
                        ),
                        Text(
                          ratings[id] == null
                              ? '0 — совсем не отвечала, 4 — полностью'
                              : ratingLabels[ratings[id]!],
                        ),
                      ],
                      const SizedBox(height: 16),
                      FilledButton(
                        onPressed: choosing ? null : next,
                        child: Text(
                          step == qs.length - 1 ? 'Проверить ответы' : 'Дальше',
                        ),
                      ),
                      TextButton(
                        onPressed:
                            choosing
                                ? null
                                : () {
                                  responses.remove(id);
                                  ratings.remove(id);
                                  next();
                                },
                        child: const Text('Пропустить вопрос'),
                      ),
                    ] else ...[
                      Text(
                        'Проверьте результат',
                        style: Theme.of(context).textTheme.headlineSmall,
                      ),
                      TextField(
                        controller: summary,
                        maxLength: 1800,
                        maxLines: 4,
                        decoration: const InputDecoration(
                          labelText: 'Моё описание ситуации',
                        ),
                        onChanged: (_) => setState(() {}),
                      ),
                      for (final e in qs.indexed)
                        ListTile(
                          title: Text(topicLabels[e.$2['id']]!),
                          subtitle: Text(responses[e.$2['id']] ?? 'Без ответа'),
                          trailing: Text(
                            ratings.containsKey(e.$2['id'])
                                ? '${ratings[e.$2['id']]}/4'
                                : '—',
                          ),
                          onTap:
                              loading
                                  ? null
                                  : () => setState(() => step = e.$1),
                        ),
                      const Text('Какая помощь сейчас нужна?'),
                      Wrap(
                        spacing: 6,
                        children: [
                          for (final e in needLabels.entries)
                            if (e.key != 'emergency' &&
                                (e.key != 'financial' ||
                                    (ratings['expenses'] ?? 4) <= 1))
                              FilterChip(
                                label: Text(e.value),
                                selected: needs.contains(e.key),
                                onSelected:
                                    loading
                                        ? null
                                        : (v) => setState(() {
                                          if (v) {
                                            needs.add(e.key);
                                          } else {
                                            needs.remove(e.key);
                                          }
                                        }),
                              ),
                        ],
                      ),
                      CheckboxListTile(
                        value: urgent,
                        title: const Text('Мне сейчас нужна экстренная помощь'),
                        onChanged:
                            loading ? null : (v) => setState(() => urgent = v!),
                      ),
                      if (urgent)
                        TextButton(
                          onPressed: () => context.push('/support/emergency'),
                          child: const Text('Открыть экстренные номера'),
                        ),
                      CheckboxListTile(
                        value: consent,
                        title: const Text(
                          'Подтверждаю описание и оценки. Сохранить их и ответы в SAYA.',
                        ),
                        onChanged:
                            loading
                                ? null
                                : (v) => setState(() => consent = v!),
                      ),
                      CheckboxListTile(
                        value: external,
                        title: const Text(
                          'Разрешаю передать внешней модели оценки, выбранные направления и отметку экстренной помощи для дополнительных рекомендаций. Текст ответов остаётся в SAYA.',
                        ),
                        onChanged:
                            loading
                                ? null
                                : (v) => setState(() => external = v!),
                      ),
                      if (error != null) Text(error!),
                      FilledButton(
                        onPressed:
                            consent &&
                                    !loading &&
                                    (responses.values.any(
                                          (v) => v.trim().isNotEmpty,
                                        ) ||
                                        ratings.isNotEmpty ||
                                        summary.text.trim().isNotEmpty ||
                                        needs.isNotEmpty ||
                                        urgent)
                                ? () => submit(data)
                                : null,
                        child: Text(
                          loading ? 'Сохранение…' : 'Сохранить результат',
                        ),
                      ),
                    ],
                    if (step > 0)
                      TextButton(
                        onPressed:
                            (loading || choosing)
                                ? null
                                : () => setState(() => step--),
                        child: const Text('Назад'),
                      ),
                  ],
                );
              },
            ),
      ),
    ),
  );
}
