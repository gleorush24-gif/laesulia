import 'dart:convert';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:http/http.dart' as http;
import 'package:latlong2/latlong.dart';
import 'auth_provider.dart';

const _apiBase = "https://laesulia-api.onrender.com";

class Bounty {
  final String id;
  final String title;
  final String description;
  final double lat;
  final double lng;
  final double rewardSbd;
  final String submitType;
  final String status;

  const Bounty({
    required this.id,
    required this.title,
    required this.description,
    required this.lat,
    required this.lng,
    required this.rewardSbd,
    required this.submitType,
    required this.status,
  });

  factory Bounty.fromJson(Map<String, dynamic> j) => Bounty(
    id:          j['id']          ?? '',
    title:       j['title']       ?? '',
    description: j['description'] ?? '',
    lat:         (j['lat']  as num).toDouble(),
    lng:         (j['lng']  as num).toDouble(),
    rewardSbd:   (j['reward_sbd'] as num).toDouble(),
    submitType:  j['submit_type'] ?? 'both',
    status:      j['status']      ?? 'open',
  );

  LatLng get latLng => LatLng(lat, lng);

  // Color based on status
  // 🔴 open, 🟡 claimed, 🟢 submitted, ✅ approved
  String get statusEmoji {
    switch (status) {
      case 'open':      return '🔴';
      case 'claimed':   return '🟡';
      case 'submitted': return '🟢';
      case 'approved':  return '✅';
      default:          return '🔴';
    }
  }
}

class BountiesState {
  final List<Bounty> bounties;
  final bool isLoading;
  final String? error;

  const BountiesState({
    this.bounties = const [],
    this.isLoading = false,
    this.error,
  });

  BountiesState copyWith({
    List<Bounty>? bounties,
    bool? isLoading,
    String? error,
  }) {
    return BountiesState(
      bounties:  bounties  ?? this.bounties,
      isLoading: isLoading ?? this.isLoading,
      error:     error     ?? this.error,
    );
  }
}

class BountiesNotifier extends Notifier<BountiesState> {
  @override
  BountiesState build() => const BountiesState();

  Future<void> fetchNearby({
    required double lat,
    required double lng,
    double radius = 50000,
  }) async {
    state = state.copyWith(isLoading: true, error: null);

    try {
      final token = ref.read(authProvider).token;

      final uri = Uri.parse(
        '$_apiBase/api/v1/bounties?lat=$lat&lng=$lng&radius=$radius',
      );

      final response = await http.get(uri, headers: {
        if (token != null) 'Authorization': 'Bearer $token',
      });

      if (response.statusCode == 200) {
        final data = jsonDecode(response.body);
        final list = (data['bounties'] as List)
            .map((e) => Bounty.fromJson(e))
            .toList();
        state = state.copyWith(bounties: list, isLoading: false);
      } else {
        state = state.copyWith(isLoading: false);
     } 
    } catch (e) {
      state = state.copyWith(isLoading: false);
    }
  }

  Future<bool> claim(String bountyId) async {
    try {
      final token = ref.read(authProvider).token;
      final response = await http.post(
        Uri.parse('$_apiBase/api/v1/bounties/$bountyId/claim'),
        headers: {'Authorization': 'Bearer $token'},
      );
      return response.statusCode == 200;
    } catch (e) {
      return false;
    }
  }
}

final bountiesProvider =
    NotifierProvider<BountiesNotifier, BountiesState>(() {
  return BountiesNotifier();
});
