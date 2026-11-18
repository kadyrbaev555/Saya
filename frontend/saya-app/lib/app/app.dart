import 'package:material_ui/material_ui.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../features/auth/session.dart';
import '../features/auth/registration.dart';
import '../features/support/support.dart';
import '../features/support/appeals.dart';
import '../features/surveys/survey.dart';
import '../features/assessments/assessment.dart';
import '../core/realtime/realtime.dart';
import '../features/assessments/weekly.dart';
import '../features/assessments/android_reminders.dart';
import 'dart:async';

final routerProvider = Provider<GoRouter>((ref) {
  final refresh = ValueNotifier<int>(0);
  ref.listen(sessionProvider, (_, next) {
    refresh.value++;
  });
  ref.onDispose(refresh.dispose);
  final router = GoRouter(
    initialLocation: '/',
    refreshListenable: refresh,
    redirect: (context, state) {
      final signedIn = ref.read(sessionProvider).asData?.value != null;
      if ((state.uri.path == '/home' ||
              (state.uri.path.startsWith('/support/') &&
                  state.uri.path != '/support/emergency') ||
              state.uri.path.startsWith('/appeals/') ||
              state.uri.path == '/history' ||
              state.uri.path == '/profile' ||
              state.uri.path == '/survey' ||
              state.uri.path.startsWith('/assessment/')) &&
          !signedIn)
        return state.uri.path == '/survey' ? '/login?next=survey' : '/login';
      if (signedIn &&
          (state.uri.path == '/login' || state.uri.path == '/register'))
        return state.uri.queryParameters['next'] == 'survey'
            ? '/survey'
            : '/home';
      return null;
    },
    routes: [
      GoRoute(path: '/home', builder: (context, state) => const SupportHome()),
      GoRoute(
        path: '/appeals',
        builder: (context, state) => const AppealsPage('public'),
      ),
      for (final scope in ['mine', 'review'])
        GoRoute(
          path: '/appeals/$scope',
          builder: (_, __) => AppealsPage(scope),
        ),
      for (final item in Direction.values)
        GoRoute(
          path: '/support/${item.path}',
          builder: (context, state) => DirectionPage(item),
        ),
      GoRoute(
        path: '/',
        builder:
            (context, state) => Scaffold(
              appBar: AppBar(title: const Text('SAYA')),
              body: Center(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    const Text('Поддержка в вашем темпе'),
                    TextButton(
                      onPressed: () async {
                        await ref.read(sessionProvider.notifier).restore();
                        if (context.mounted) context.go('/home');
                      },
                      child: const Text('Продолжить сессию'),
                    ),
                    const SizedBox(height: 24),
                    FilledButton(
                      onPressed: () => context.go('/login'),
                      child: const Text('Войти'),
                    ),
                    TextButton(
                      onPressed: () => context.go('/register'),
                      child: const Text('Создать аккаунт'),
                    ),
                    TextButton(
                      onPressed: () => context.push('/appeals'),
                      child: const Text('Помочь человеку'),
                    ),
                    TextButton(
                      onPressed: () => context.push('/support/emergency'),
                      child: const Text('Экстренная помощь'),
                    ),
                  ],
                ),
              ),
            ),
      ),
      GoRoute(path: '/login', builder: (context, state) => const AuthForm()),
      GoRoute(
        path: '/register',
        builder: (context, state) => const RegistrationPage(),
      ),
      GoRoute(path: '/survey', builder: (context, state) => const SurveyPage()),
      GoRoute(
        path: '/assessment/:id',
        builder:
            (context, state) => AssessmentPage(id: state.pathParameters['id']!),
      ),
      GoRoute(
        path: '/history',
        builder: (context, state) => const WeeklyDashboard(),
      ),
      GoRoute(path: '/profile', builder: (context, state) => const Profile()),
    ],
  );
  ref.onDispose(router.dispose);
  return router;
});

class SayaApp extends ConsumerStatefulWidget {
  const SayaApp({super.key});
  @override
  ConsumerState<SayaApp> createState() => _SayaAppState();
}

class _SayaAppState extends ConsumerState<SayaApp> {
  @override
  void initState() {
    super.initState();
    if (AndroidReminders.supported)
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) return;
        unawaited(
          AndroidReminders.instance.initialize(
            ref.read(apiProvider),
            () =>
                mounted
                    ? (ref.read(sessionProvider).asData?.value?['id']
                        as String?)
                    : null,
            () async {
              if (!mounted) return;
              if (ref.read(sessionProvider).asData?.value == null)
                await ref.read(sessionProvider.notifier).restore();
              if (!mounted) return;
              if (ref.read(sessionProvider).asData?.value == null) {
                ref.read(routerProvider).go('/login?next=survey');
                return;
              }
              try {
                final h = await ref
                    .read(apiProvider)
                    .request('/weekly/history');
                if (mounted)
                  ref
                      .read(routerProvider)
                      .go(h['reminder_due'] == true ? '/survey' : '/history');
              } catch (_) {
                if (mounted) ref.read(routerProvider).go('/history');
              }
            },
          ),
        );
      });
  }

  @override
  void dispose() {
    if (AndroidReminders.supported) AndroidReminders.instance.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (AndroidReminders.supported)
      ref.listen(sessionProvider, (previous, next) {
        final owner = next.asData?.value?['id'] as String?;
        if (owner != null)
          unawaited(
            AndroidReminders.instance
                .reconnect(ref.read(apiProvider), owner)
                .catchError((Object _) {}),
          );
      });
    return MaterialApp.router(
      title: 'SAYA',
      theme: ThemeData(useMaterial3: true, colorSchemeSeed: Colors.teal),
      routerConfig: ref.watch(routerProvider),
    );
  }
}

class AuthForm extends ConsumerStatefulWidget {
  const AuthForm({super.key});
  @override
  ConsumerState<AuthForm> createState() => _AuthFormState();
}

class _AuthFormState extends ConsumerState<AuthForm> {
  final email = TextEditingController(), password = TextEditingController();
  @override
  void dispose() {
    email.dispose();
    password.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(sessionProvider);
    return Scaffold(
      appBar: AppBar(title: Text('Войти')),
      body: Center(
        child: SingleChildScrollView(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 440),
            child: Padding(
              padding: const EdgeInsets.all(24),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  TextField(
                    controller: email,
                    keyboardType: TextInputType.emailAddress,
                    decoration: const InputDecoration(labelText: 'Email'),
                  ),
                  const SizedBox(height: 16),
                  TextField(
                    controller: password,
                    obscureText: true,
                    enableSuggestions: false,
                    autocorrect: false,
                    decoration: const InputDecoration(
                      labelText: 'Пароль (от 12 символов)',
                    ),
                  ),
                  const SizedBox(height: 24),
                  if (state.hasError)
                    const Text(
                      'Не удалось войти. Проверьте данные и соединение.',
                    ),
                  FilledButton(
                    onPressed:
                        state.isLoading
                            ? null
                            : () async {
                              await ref
                                  .read(sessionProvider.notifier)
                                  .authenticate(
                                    'login',
                                    email.text,
                                    password.text,
                                  );
                            },
                    child:
                        state.isLoading
                            ? const CircularProgressIndicator()
                            : const Text('Продолжить'),
                  ),
                  TextButton(
                    onPressed: () => context.go('/register'),
                    child: Text('Создать аккаунт'),
                  ),
                  TextButton(
                    onPressed: () => context.push('/appeals'),
                    child: const Text('Помочь человеку'),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class Profile extends ConsumerWidget {
  const Profile({super.key});
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final realtime = ref.watch(realtimeProvider);
    final profile = ref.watch(profileProvider);
    final user =
        profile.asData?.value ?? ref.watch(sessionProvider).asData?.value;
    return Scaffold(
      appBar: AppBar(
        title: const Text('Профиль'),
        actions: [
          IconButton(
            tooltip: 'На главную',
            onPressed: () => context.go('/home'),
            icon: const Icon(Icons.home_outlined),
          ),
        ],
      ),
      body: Center(
        child: ListView(
          shrinkWrap: true,
          padding: const EdgeInsets.all(16),
          children: [
            const WeeklyInvite(),
            if (user?['role'] == 'ADMIN')
              TextButton(
                onPressed: () => context.push('/appeals/review'),
                child: const Text('Модерация анкет'),
              ),
            if (((ref.watch(appealsProvider('mine')).asData?.value['items']
                        as List?) ??
                    [])
                .isNotEmpty)
              TextButton(
                onPressed: () => context.push('/appeals/mine'),
                child: const Text('Моя анкета'),
              ),
            Text(user?['email'] as String? ?? ''),
            const SizedBox(height: 16),
            ref
                .watch(latestSurveyProvider)
                .when(
                  data:
                      (id) =>
                          id == null || id.isEmpty
                              ? const Text('Вы ещё не проходили опрос')
                              : OutlinedButton(
                                onPressed:
                                    () => context.push('/assessment/$id'),
                                child: const Text('Последний результат'),
                              ),
                  error:
                      (_, __) => TextButton(
                        onPressed: () => ref.invalidate(latestSurveyProvider),
                        child: const Text('Повторить загрузку результата'),
                      ),
                  loading: () => const LinearProgressIndicator(),
                ),
            FilledButton(
              onPressed: () => context.go('/survey'),
              child: const Text('Пройти опрос о неделе'),
            ),
            OutlinedButton(
              onPressed: () => context.push('/history'),
              child: const Text('Динамика по неделям'),
            ),
            Text(
              realtime.asData?.value['event'] == 'demo.completed'
                  ? 'Данные обновлены'
                  : realtime.asData?.value['status'] == 'connected'
                  ? 'Подключено'
                  : 'Подключение…',
            ),
            TextButton(
              onPressed: () {
                ref.read(sessionProvider.notifier).logout();
              },
              child: const Text('Выйти'),
            ),
          ],
        ),
      ),
    );
  }
}
