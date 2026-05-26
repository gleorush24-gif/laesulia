import 'dart:convert';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:http/http.dart' as http;
import 'package:latlong2/latlong.dart';
import 'auth_provider.dart';

const _apiBase = "http://localhost:8081";

class WanLocation {
  final String id;
  final String name;
  final String localName;
  final String category;
  final double lat;
  final double lng;
  final String addressCode;
  final int upvotes;
  final bool verified;

  const WanLocation({
    required this.id,
    required this.name,
    required this.localName,
    required this.category,
    required this.lat,
    required this.lng,
    required this.addressCode,
    required this.upvotes,
    required this.verified,
  });

  factory WanLocation.fromJson(Map<String, dynamic> j) => WanLocation(
    id:          j['id']           ?? '',
    name:        j['name']         ?? '',
    localName:   j['local_name']   ?? '',
    category:    j['category']     ?? 'place',
    lat:         (j['lat']  as num).toDouble(),
    lng:         (j['lng']  as num).toDouble(),
    addressCode: j['address_code'] ?? '',
    upvotes:     j['upvotes']      ?? 0,
    verified:    j['verified']     ?? false,
  );

  LatLng get latLng => LatLng(lat, lng);

  String get displayName =>
    localName.isNotEmpty ? '$name ($localName)' : name;

  String get emoji {
    const map = {
      'market': '🏪', 'beach': '🏖️', 'creek': '🌊',
      'hill': '⛰️',  'village': '🏡', 'church': '⛪',
      'school': '🏫', 'clinic': '🏥', 'road': '🛣️',
      'sacred': '🪨', 'government': '🏛️', 'business': '🏬',
    };
    return map[category] ?? '📍';
  }
}

class LocationsState {
  final List<WanLocation> locations;
  final bool isLoading;
  final String? error;

  const LocationsState({
    this.locations = const [],
    this.isLoading = false,
    this.error,
  });

  LocationsState copyWith({
    List<WanLocation>? locations,
    bool? isLoading,
    String? error,
  }) {
    return LocationsState(
      locations: locations ?? this.locations,
      isLoading: isLoading ?? this.isLoading,
      error:     error     ?? this.error,
    );
  }
}

class LocationsNotifier extends Notifier<LocationsState> {
  @override
  LocationsState build() => const LocationsState();

  Future<void> fetchNearby({
    required double lat,
    required double lng,
    double radius = 10000,
  }) async {
    state = state.copyWith(isLoading: true, error: null);

    try {
      final token = ref.read(authProvider).token;
      final uri   = Uri.parse(
        '$_apiBase/api/v1/locations?lat=$lat&lng=$lng&radius=$radius',
      );

      final response = await http.get(uri, headers: {
        if (token != null) 'Authorization': 'Bearer $token',
      });

      if (response.statusCode == 200) {
        final data = jsonDecode(response.body);
        final list = (data['locations'] as List)
            .map((e) => WanLocation.fromJson(e))
            .toList();
        state = state.copyWith(locations: list, isLoading: false);
      } else {
        state = state.copyWith(
          isLoading: false,
          error: 'Failed to load locations',
        );
      }
    } catch (e) {
      state = state.copyWith(
        isLoading: false,
        error: 'Network error: $e',
      );
    }
  }
}

final locationsProvider =
    NotifierProvider<LocationsNotifier, LocationsState>(() {
  return LocationsNotifier();
});