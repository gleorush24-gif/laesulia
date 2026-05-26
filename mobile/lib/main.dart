import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:google_fonts/google_fonts.dart';

import 'screens/map_screen.dart';
import 'screens/login_screen.dart';

void main() {
  runApp(const ProviderScope(child: LaesuliaApp()));
}

const kApiBase = 'http://localhost:8081';

final _router = GoRouter(
  initialLocation: '/login',
  routes: [
    GoRoute(path: '/login', builder: (context, state) => const LoginScreen()),
    ShellRoute(
      builder: (context, state, child) => AppShell(child: child),
      routes: [
        GoRoute(path: '/map',       builder: (context, state) => const MapScreen()),
        GoRoute(path: '/places',    builder: (context, state) => const MapScreen()),
        GoRoute(path: '/community', builder: (context, state) => const MapScreen()),
      ],
    ),
  ],
);

class LaesuliaApp extends StatelessWidget {
  const LaesuliaApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp.router(
      title: 'Laesulia',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        useMaterial3: true,
        colorScheme: ColorScheme.fromSeed(
          seedColor: const Color(0xFF0066CC),
        ),
        textTheme: GoogleFonts.outfitTextTheme(),
      ),
      routerConfig: _router,
    );
  }
}

class AppShell extends ConsumerWidget {
  final Widget child;
  const AppShell({super.key, required this.child});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final location = GoRouterState.of(context).uri.toString();

    int selectedIndex = 0;
    if (location.startsWith('/places'))    selectedIndex = 1;
    if (location.startsWith('/community')) selectedIndex = 2;

    return Scaffold(
      body: child,
      bottomNavigationBar: NavigationBar(
        selectedIndex: selectedIndex,
        onDestinationSelected: (i) {
          const paths = ['/map', '/places', '/community'];
          context.go(paths[i]);
        },
        destinations: const [
          NavigationDestination(
            icon: Icon(Icons.map_outlined),
            selectedIcon: Icon(Icons.map),
            label: 'Map',
          ),
          NavigationDestination(
            icon: Icon(Icons.place_outlined),
            selectedIcon: Icon(Icons.place),
            label: 'Places',
          ),
          NavigationDestination(
            icon: Icon(Icons.people_outlined),
            selectedIcon: Icon(Icons.people),
            label: 'Community',
          ),
        ],
      ),
    );
  }
}
