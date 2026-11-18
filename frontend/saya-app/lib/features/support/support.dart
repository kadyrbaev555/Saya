import 'housing.dart';
import 'jobs.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../auth/session.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';
import 'package:url_launcher/url_launcher.dart';

final externalLinkProvider = Provider<Future<bool> Function(Uri)>(
  (ref) => (uri) => launchUrl(uri),
);

enum Direction {
  housing('housing', 'Жильё', 'Посуточная аренда', Icons.home_outlined),
  jobs('jobs', 'Поиск работы', 'Вакансии по вашим навыкам', Icons.work_outline),
  psychologist(
    'psychologist',
    'Психолог',
    'Подготовить обращение',
    Icons.forum_outlined,
  ),
  social(
    'social',
    'Социальная поддержка',
    'Волонтёрские организации',
    Icons.people_outline,
  ),
  emergency(
    'emergency',
    'Экстренная помощь',
    'Номера служб Казахстана',
    Icons.phone_outlined,
  );

  const Direction(this.path, this.title, this.description, this.icon);
  final String path, title, description;
  final IconData icon;
}

class SupportHome extends StatelessWidget {
  const SupportHome({super.key});

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(
      title: const Text('SAYA'),
      actions: [
        TextButton.icon(
          onPressed: () => context.push('/profile'),
          icon: const Icon(Icons.person_outline),
          label: const Text('Профиль'),
        ),
      ],
    ),
    body: SafeArea(
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 800),
          child: ListView(
            padding: const EdgeInsets.all(20),
            children: [
              Text(
                'Какая помощь вам нужна?',
                style: Theme.of(context).textTheme.headlineSmall,
              ),
              const SizedBox(height: 16),
              for (final item in Direction.values)
                Card(
                  child: ListTile(
                    leading: Icon(item.icon),
                    title: Text(item.title),
                    subtitle: Text(item.description),
                    trailing: const Icon(Icons.chevron_right),
                    onTap: () => context.push('/support/${item.path}'),
                  ),
                ),
            ],
          ),
        ),
      ),
    ),
  );
}

class PublicAppeals extends StatelessWidget {
  const PublicAppeals({super.key});
  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(title: const Text('Помочь человеку')),
    body: const Center(
      child: Padding(
        padding: EdgeInsets.all(24),
        child: Text('Пока нет опубликованных анкет для помощи.'),
      ),
    ),
  );
}

class DirectionPage extends ConsumerStatefulWidget {
  const DirectionPage(this.direction, {super.key});
  final Direction direction;
  @override
  ConsumerState<DirectionPage> createState() => _DirectionPageState();
}

class _DirectionPageState extends ConsumerState<DirectionPage> {
  final draft = TextEditingController();
  bool edited = false;
  bool filled = false;
  @override
  void initState() {
    super.initState();
    if (widget.direction == Direction.psychologist ||
        widget.direction == Direction.social)
      loadSummary();
  }

  Future<void> loadSummary() async {
    try {
      final api = ref.read(apiProvider);
      final id = (await api.request('/surveys/latest'))['id'] as String?;
      if (!mounted || id == null || id.isEmpty) return;
      final result = await api.request('/assessments/$id');
      final summary = result['summary'] as String?;
      if (mounted &&
          !edited &&
          result['status'] == 'completed' &&
          summary != null &&
          summary.isNotEmpty) {
        setState(() {
          draft.text = summary;
          filled = true;
        });
      }
    } catch (_) {
      /* Manual entry remains available when the result cannot load. */
    }
  }

  @override
  void dispose() {
    draft.dispose();
    super.dispose();
  }

  Future<void> open(Uri uri) async {
    try {
      if (await ref.read(externalLinkProvider)(uri)) return;
    } on PlatformException {
      // Some devices have no application registered for telephone links.
    }
    if (mounted)
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Не удалось открыть ${uri.toString()}')),
      );
  }

  Widget link(String label, Uri Function() uri) => FilledButton.tonalIcon(
    onPressed: () => open(uri()),
    icon: const Icon(Icons.open_in_new),
    label: Text(label),
  );

  @override
  Widget build(BuildContext context) {
    final direction = widget.direction;
    return Scaffold(
      appBar: AppBar(title: Text(direction.title)),
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 680),
          child: ListView(
            padding: const EdgeInsets.all(24),
            children: [
              if (direction == Direction.housing) HousingSearch(open: open),
              if (direction == Direction.jobs) JobSearch(open: open),
              if (direction == Direction.social ||
                  direction == Direction.psychologist) ...[
                TextField(
                  controller: draft,
                  onChanged: (_) => edited = true,
                  maxLines: 4,
                  decoration: const InputDecoration(
                    labelText: 'Что вы хотите рассказать?',
                    alignLabelWithHint: true,
                  ),
                ),
                const SizedBox(height: 8),
                TextButton.icon(
                  icon: const Icon(Icons.copy),
                  label: const Text('Скопировать обращение'),
                  onPressed: () async {
                    final message = draft.text.trim();
                    if (message.isEmpty) {
                      ScaffoldMessenger.of(context).showSnackBar(
                        const SnackBar(
                          content: Text('Введите текст обращения.'),
                        ),
                      );
                      return;
                    }
                    await Clipboard.setData(ClipboardData(text: message));
                    if (context.mounted)
                      ScaffoldMessenger.of(context).showSnackBar(
                        const SnackBar(content: Text('Обращение скопировано.')),
                      );
                  },
                ),
                const SizedBox(height: 16),
                if (direction == Direction.social) ...[
                  const Text(
                    'Birgemiz · Национальный фронт-офис волонтёров. Можно узнать, в какую организацию Алматы обратиться. Вставьте подготовленный текст в чат перед отправкой.',
                  ),
                  link(
                    'Написать Birgemiz в Telegram',
                    () => Uri.parse('https://t.me/BIRGEMIZ2020'),
                  ),
                  link(
                    'Региональные контакты Qazvolunteer',
                    () => Uri.parse('https://qazvolunteer.kz/ru/contacts'),
                  ),
                  link(
                    'Добрые дела Алматы · помощь одиноким пенсионерам',
                    () => Uri.parse('https://dobrye-dela.kz/'),
                  ),
                ],
                if (direction == Direction.social)
                  link(
                    'Найти организацию на Qazvolunteer',
                    () => Uri.parse('https://qazvolunteer.kz/ru'),
                  ),
                if (direction == Direction.psychologist) ...[
                  const Text(
                    'Алматы · Центр психического здоровья. Бесплатная анонимная психологическая помощь круглосуточно.',
                  ),
                  OutlinedButton.icon(
                    onPressed: () => open(Uri.parse('tel:1303')),
                    icon: const Icon(Icons.phone_outlined),
                    label: const Text('Набрать 1303'),
                  ),
                  link(
                    'Информация городской службы',
                    () => Uri.parse(
                      'https://www.gov.kz/org/almaty-densaulyk/press/news/details/1220242?lang=ru',
                    ),
                  ),

                  const Text(
                    'Линия 150 оказывает бесплатную психологическую помощь детям, молодёжи и их семьям.',
                  ),
                  OutlinedButton.icon(
                    onPressed: () => open(Uri.parse('tel:150')),
                    icon: const Icon(Icons.phone_outlined),
                    label: const Text('Набрать 150'),
                  ),
                  const Divider(),
                  Text(
                    filled
                        ? 'Вы можете изменить описание из последнего опроса перед обращением.'
                        : 'В профиле можно пройти первоначальный опрос и получить рекомендации.',
                  ),
                  TextButton(
                    onPressed: () => context.push('/survey'),
                    child: const Text('Перейти к опросу'),
                  ),
                ],
              ],
              if (direction == Direction.emergency)
                for (final (number, label) in const [
                  ('112', 'Единая экстренная служба'),
                  ('101', 'Пожарные и спасатели'),
                  ('102', 'Полиция'),
                  ('103', 'Скорая медицинская помощь'),
                  ('104', 'Аварийная газовая служба'),
                ])
                  Card(
                    child: ListTile(
                      title: Text(number),
                      subtitle: Text(label),
                      trailing: const Icon(Icons.phone_outlined),
                      onTap: () => open(Uri.parse('tel:$number')),
                    ),
                  ),
            ],
          ),
        ),
      ),
    );
  }
}
