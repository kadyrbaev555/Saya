import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';
import 'package:url_launcher/url_launcher.dart';
import '../auth/session.dart';

final appealsProvider = FutureProvider.autoDispose
    .family<Map<String, dynamic>, String>((ref, scope) {
      ref.watch(sessionProvider.select((s) => s.asData?.value?['id']));
      return ref
          .watch(apiProvider)
          .request(scope == 'public' ? '/appeals' : '/appeals/$scope');
    });

const states = {
  'pending': 'На модерации',
  'published': 'Опубликована',
  'declined': 'Отклонена',
  'withdrawn': 'Снята',
  'expired': 'Скрыта после нового опроса',
};

class AppealsPage extends ConsumerStatefulWidget {
  final String scope;
  const AppealsPage(this.scope, {super.key});
  @override
  ConsumerState<AppealsPage> createState() => _AppealsPageState();
}

class _AppealsPageState extends ConsumerState<AppealsPage> {
  bool busy = false;
  Future<void> act(String id, String action, {int? amount}) async {
    setState(() => busy = true);
    try {
      final data = await ref
          .read(apiProvider)
          .request(
            '/appeals/$id/$action',
            body: {if (amount != null) 'amount': amount},
          );
      if (action == 'checkout' || action == 'payout-card') {
        final opened = await launchUrl(
          Uri.parse(data['url'] as String),
          webOnlyWindowName: '_self',
        );
        if (!opened) throw StateError('checkout unavailable');
      } else if (mounted) {
        ref.invalidate(appealsProvider);
        if (action == 'report')
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('Жалоба передана на проверку.')),
          );
      }
    } catch (_) {
      if (mounted)
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text(
              'Действие не выполнено. Обновите список и попробуйте ещё раз.',
            ),
          ),
        );
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final scope = widget.scope;
    return Scaffold(
      appBar: AppBar(
        title: Text(
          scope == 'review'
              ? 'Модерация анкет'
              : scope == 'mine'
              ? 'Моя анкета'
              : 'Помочь человеку',
        ),
        actions: [
          IconButton(
            tooltip: 'Обновить',
            icon: const Icon(Icons.refresh),
            onPressed: () => ref.invalidate(appealsProvider(scope)),
          ),
        ],
      ),
      body: ref
          .watch(appealsProvider(scope))
          .when(
            loading: () => const Center(child: CircularProgressIndicator()),
            error:
                (_, __) => const Center(
                  child: Text(
                    'Не удалось загрузить анкеты. Попробуйте обновить список.',
                  ),
                ),
            data: (data) {
              final items = data['items'] as List? ?? [];
              if (items.isEmpty)
                return Center(
                  child: Text(
                    scope == 'public'
                        ? 'Пока нет опубликованных анкет для помощи.'
                        : scope == 'review'
                        ? 'Очередь проверки пуста.'
                        : 'Вы пока не подавали анкету.',
                  ),
                );
              return ListView(
                padding: const EdgeInsets.all(16),
                children: [
                  for (final a in items)
                    Card(
                      child: Padding(
                        padding: const EdgeInsets.all(16),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(a['description'] as String),
                            if (scope != 'public')
                              Text(states[a['state']] ?? ''),
                            if (scope == 'mine' &&
                                data['checkout_available'] == true &&
                                ['pending', 'published'].contains(a['state']))
                              OutlinedButton(
                                onPressed:
                                    busy
                                        ? null
                                        : () => act(a['id'], 'payout-card'),
                                child: Text(
                                  a['card_ready'] == true
                                      ? 'Изменить карту у Freedom Pay'
                                      : 'Привязать тестовую карту у Freedom Pay',
                                ),
                              ),
                            if (scope == 'mine' && a['state'] != 'withdrawn')
                              TextButton(
                                onPressed:
                                    busy
                                        ? null
                                        : () => act(a['id'], 'withdraw'),
                                child: const Text('Снять анкету'),
                              ),
                            if (scope == 'review') ...[
                              if (a['reported'] == true)
                                const Text('Поступила жалоба'),
                              Wrap(
                                children: [
                                  for (final action in ['approve', 'decline'])
                                    TextButton(
                                      onPressed:
                                          busy
                                              ? null
                                              : () => act(a['id'], action),
                                      child: Text(
                                        action == 'approve'
                                            ? 'Одобрить'
                                            : 'Отклонить',
                                      ),
                                    ),
                                ],
                              ),
                            ],
                            if (scope == 'public') ...[
                              if (data['checkout_available'] == true &&
                                  a['card_ready'] == true)
                                Wrap(
                                  children: [
                                    for (final amount in [500, 1000, 2000])
                                      OutlinedButton(
                                        onPressed:
                                            busy
                                                ? null
                                                : () => act(
                                                  a['id'],
                                                  'checkout',
                                                  amount: amount,
                                                ),
                                        child: Text(
                                          'Тестовый платёж · $amount ₸',
                                        ),
                                      ),
                                  ],
                                )
                              else
                                const Text(
                                  'Тестовый платёж для этой анкеты пока недоступен.',
                                ),
                              TextButton(
                                onPressed:
                                    busy ? null : () => act(a['id'], 'report'),
                                child: const Text('Пожаловаться'),
                              ),
                            ],
                          ],
                        ),
                      ),
                    ),
                ],
              );
            },
          ),
    );
  }
}

Future<void> editAppeal(
  BuildContext context,
  WidgetRef ref,
  String assessment,
  String summary,
) async {
  final text = await showDialog<String>(
    context: context,
    builder: (_) => _AppealEditor(summary),
  );
  if (text == null || !context.mounted) return;
  try {
    await ref
        .read(apiProvider)
        .request(
          '/appeals/$assessment/submit',
          body: {'description': text, 'consent': true},
        );
    if (!context.mounted) return;
    ref.invalidate(appealsProvider);
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('Анкета отправлена на модерацию.')),
    );
  } catch (_) {
    if (context.mounted)
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text(
            'Анкету не удалось подать. Проверьте последнюю рекомендацию в профиле.',
          ),
        ),
      );
  }
}

class _AppealEditor extends StatefulWidget {
  final String initial;
  const _AppealEditor(this.initial);
  @override
  State<_AppealEditor> createState() => _AppealEditorState();
}

class _AppealEditorState extends State<_AppealEditor> {
  late final controller = TextEditingController(text: widget.initial);
  bool consent = false;
  @override
  void dispose() {
    controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => AlertDialog(
    title: const Text('Анкета для помощи'),
    content: SingleChildScrollView(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          TextField(
            controller: controller,
            maxLength: 1200,
            minLines: 3,
            maxLines: 6,
            onChanged: (_) => setState(() => consent = false),
            decoration: const InputDecoration(
              labelText: 'Публичное описание',
              helperText: 'Уберите контакты и реквизиты карты.',
              helperMaxLines: 2,
            ),
          ),
          CheckboxListTile(
            value: consent,
            onChanged: (v) => setState(() => consent = v ?? false),
            title: const Text(
              'Согласен опубликовать этот текст для всех посетителей после модерации.',
            ),
          ),
        ],
      ),
    ),
    actions: [
      TextButton(
        onPressed: () => Navigator.pop(context),
        child: const Text('Отмена'),
      ),
      FilledButton(
        onPressed:
            consent &&
                    controller.text.trim().isNotEmpty &&
                    controller.text.runes.length <= 1200
                ? () => Navigator.pop(context, controller.text.trim())
                : null,
        child: const Text('Отправить на проверку'),
      ),
    ],
  );
}
