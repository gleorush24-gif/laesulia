import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:http/http.dart' as http;
import '../providers/auth_provider.dart';

const _apiBase = 'https://laesulia-api.onrender.com';

class LoginScreen extends ConsumerStatefulWidget {
  const LoginScreen({super.key});
  @override
  ConsumerState<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends ConsumerState<LoginScreen> {
  final _emailCtrl    = TextEditingController();
  final _passwordCtrl = TextEditingController();
  bool _isLoading = false;
  String? _error;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: const Color(0xFF0A1628),
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: Container(
            width: 400,
            padding: const EdgeInsets.all(36),
            decoration: BoxDecoration(
              color: Colors.white,
              borderRadius: BorderRadius.circular(20),
              boxShadow: [BoxShadow(color: Colors.black.withOpacity(0.3), blurRadius: 30)],
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('🗺️ Laesulia',
                  style: GoogleFonts.fraunces(fontSize: 28, fontWeight: FontWeight.w700)),
                const SizedBox(height: 4),
                RichText(text: TextSpan(
                  style: GoogleFonts.outfit(fontSize: 14, color: Colors.grey[600]),
                  children: [
                    const TextSpan(text: '"'),
                    TextSpan(text: 'to follow',
                      style: GoogleFonts.outfit(color: const Color(0xFF0066CC), fontWeight: FontWeight.w600)),
                    const TextSpan(text: '" — Toobaita, Malaita'),
                  ],
                )),
                const SizedBox(height: 32),
                Text('Email', style: GoogleFonts.outfit(fontWeight: FontWeight.w600, fontSize: 12, color: Colors.grey[600], letterSpacing: 0.5)),
                const SizedBox(height: 6),
                TextField(
                  controller: _emailCtrl,
                  decoration: InputDecoration(
                    hintText: 'your@email.com',
                    border: OutlineInputBorder(borderRadius: BorderRadius.circular(10)),
                    focusedBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(10),
                      borderSide: const BorderSide(color: Color(0xFF0066CC), width: 2)),
                  ),
                  keyboardType: TextInputType.emailAddress,
                ),
                const SizedBox(height: 16),
                Text('Password', style: GoogleFonts.outfit(fontWeight: FontWeight.w600, fontSize: 12, color: Colors.grey[600], letterSpacing: 0.5)),
                const SizedBox(height: 6),
                TextField(
                  controller: _passwordCtrl,
                  obscureText: true,
                  decoration: InputDecoration(
                    hintText: '••••••••',
                    border: OutlineInputBorder(borderRadius: BorderRadius.circular(10)),
                    focusedBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(10),
                      borderSide: const BorderSide(color: Color(0xFF0066CC), width: 2)),
                  ),
                  onSubmitted: (_) => _login(),
                ),
                if (_error != null) ...[
                  const SizedBox(height: 12),
                  Text(_error!, style: GoogleFonts.outfit(color: Colors.red, fontSize: 13)),
                ],
                const SizedBox(height: 24),
                SizedBox(
                  width: double.infinity,
                  child: ElevatedButton(
                    onPressed: _isLoading ? null : _login,
                    style: ElevatedButton.styleFrom(
                      backgroundColor: const Color(0xFF0066CC),
                      padding: const EdgeInsets.symmetric(vertical: 14),
                      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
                    ),
                    child: _isLoading
                      ? const SizedBox(width: 20, height: 20,
                          child: CircularProgressIndicator(color: Colors.white, strokeWidth: 2))
                      : Text('Sign In', style: GoogleFonts.outfit(color: Colors.white, fontWeight: FontWeight.w700, fontSize: 16)),
                  ),
                ),
                const SizedBox(height: 12),
                SizedBox(
                  width: double.infinity,
                  child: TextButton(
                    onPressed: _isLoading ? null : _register,
                    child: Text('New user? Register',
                      style: GoogleFonts.outfit(color: const Color(0xFF0066CC))),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Future<void> _login() async {
    setState(() { _isLoading = true; _error = null; });
    try {
      final res = await http.post(
        Uri.parse('$_apiBase/api/v1/auth/login'),
        headers: {'Content-Type': 'application/json'},
        body: jsonEncode({'email': _emailCtrl.text.trim(), 'password': _passwordCtrl.text}),
      );
      final data = jsonDecode(res.body);
      if (res.statusCode == 200) {
        await ref.read(authProvider.notifier).setAuth(data['token'], data['user_id'], data['username']);
        if (mounted) context.go('/map');
      } else {
        setState(() => _error = data['error'] ?? 'Login failed');
      }
    } catch (e) {
      setState(() => _error = 'Error: $e');
    } finally {
      setState(() => _isLoading = false);
    }
  }

  Future<void> _register() async {
    setState(() { _isLoading = true; _error = null; });
    try {
      final email    = _emailCtrl.text.trim();
      final username = email.split('@').first;
      final res = await http.post(
        Uri.parse('$_apiBase/api/v1/auth/register'),
        headers: {'Content-Type': 'application/json'},
        body: jsonEncode({'username': username, 'email': email, 'password': _passwordCtrl.text}),
      );
      final data = jsonDecode(res.body);
      if (res.statusCode == 201) {
        await ref.read(authProvider.notifier).setAuth(data['token'], data['user_id'], data['username']);
        if (mounted) context.go('/map');
      } else {
        setState(() => _error = data['error'] ?? 'Register failed');
      }
    } catch (e) {
      setState(() => _error = 'Error: $e');
    } finally {
      setState(() => _isLoading = false);
    }
  }
}
