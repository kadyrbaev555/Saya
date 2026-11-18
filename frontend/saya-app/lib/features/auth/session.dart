import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/network/api.dart';
import '../assessments/android_reminders.dart';

final apiProvider = Provider<Api>((ref) {
  final api = Api();
  ref.onDispose(api.close);
  return api;
});
final sessionProvider =
    NotifierProvider<Session, AsyncValue<Map<String, dynamic>?>>(Session.new);

class Session extends Notifier<AsyncValue<Map<String, dynamic>?>> {
  @override
  AsyncValue<Map<String, dynamic>?> build() => const AsyncData(null);
  Future<void> restore() async {
    if (state.isLoading) return;
    state = const AsyncLoading();
    try {
      await ref.read(apiProvider).refresh();
      final profile = await ref.read(apiProvider).request('/users/me');
      if (ref.mounted) state = AsyncData(profile);
    } catch (_) {
      if (ref.mounted) state = const AsyncData(null);
    }
  }

  Future<void> authenticate(
    String action,
    String email,
    String password,
  ) async {
    state = const AsyncLoading();
    final next = await AsyncValue.guard(() async {
      final api = ref.read(apiProvider);
      await api.authenticate(action, email, password);
      return api.request('/users/me');
    });
    if (ref.mounted) state = next;
  }

  Future<void> confirmRegistration(String id, String code) async {
    final api = ref.read(apiProvider);
    final profile = await api.confirmRegistration(id, code);
    if (ref.mounted) state = AsyncData(profile);
  }

  Future<void> logout() async {
    state = const AsyncLoading();
    final next = await AsyncValue.guard(() async {
      final api = ref.read(apiProvider);
      await AndroidReminders.instance.signOut(api);
      await api.logout();
      return null;
    });
    if (ref.mounted) state = next;
  }

  void clearLocal() {
    ref.read(apiProvider).access = null;
    state = const AsyncData(null);
  }
}
