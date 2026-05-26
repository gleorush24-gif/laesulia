import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

class AuthState {
  final String? token;
  final String? userId;
  final String? username;

  const AuthState({this.token, this.userId, this.username});

  bool get isLoggedIn => token != null;

  AuthState copyWith({String? token, String? userId, String? username}) {
    return AuthState(
      token:    token    ?? this.token,
      userId:   userId   ?? this.userId,
      username: username ?? this.username,
    );
  }
}

class AuthNotifier extends Notifier<AuthState> {
  @override
  AuthState build() {
    _loadFromStorage();
    return const AuthState();
  }

  Future<void> _loadFromStorage() async {
    final prefs    = await SharedPreferences.getInstance();
    final token    = prefs.getString('token');
    final userId   = prefs.getString('user_id');
    final username = prefs.getString('username');
    if (token != null) {
      state = AuthState(token: token, userId: userId, username: username);
    }
  }

  Future<void> setAuth(String token, String userId, String username) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('token',    token);
    await prefs.setString('user_id',  userId);
    await prefs.setString('username', username);
    state = AuthState(token: token, userId: userId, username: username);
  }

  Future<void> logout() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.clear();
    state = const AuthState();
  }
}

final authProvider = NotifierProvider<AuthNotifier, AuthState>(() {
  return AuthNotifier();
});