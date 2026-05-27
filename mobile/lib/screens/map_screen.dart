
import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:http/http.dart' as http;
import 'package:latlong2/latlong.dart';
import 'upload_screen.dart';
import '../providers/locations_provider.dart';
import '../providers/bounties_provider.dart';
import '../providers/auth_provider.dart';

const _honiaraCenter = LatLng(-9.4313, 160.0521);
const _apiBase = 'https://laesulia-api.onrender.com';

class MapScreen extends ConsumerStatefulWidget {
  const MapScreen({super.key});
  @override
  ConsumerState<MapScreen> createState() => _MapScreenState();
}

class _MapScreenState extends ConsumerState<MapScreen> {
  final _mapController  = MapController();
  bool  _bountyDropMode = false;

  @override
  void initState() {
    super.initState();
    Future.microtask(() {
      ref.read(locationsProvider.notifier).fetchNearby(
        lat: _honiaraCenter.latitude, lng: _honiaraCenter.longitude);
      ref.read(bountiesProvider.notifier).fetchNearby(
        lat: _honiaraCenter.latitude, lng: _honiaraCenter.longitude);
    });
  }

  @override
  Widget build(BuildContext context) {
    final locState    = ref.watch(locationsProvider);
    final bountyState = ref.watch(bountiesProvider);

    return Scaffold(
      body: Stack(
        children: [
          FlutterMap(
            mapController: _mapController,
            options: MapOptions(
              initialCenter: _honiaraCenter,
              initialZoom: 13.0,
              minZoom: 3.0,
              maxZoom: 19.0,
              interactionOptions: const InteractionOptions(flags: InteractiveFlag.all),
              onTap: (tapPosition, point) {
                if (_bountyDropMode) {
                  setState(() => _bountyDropMode = false);
                  _showDropBountySheet(point.latitude, point.longitude);
                }
              },
              onMapEvent: (event) {
                if (event is MapEventMoveEnd) {
                  ref.read(locationsProvider.notifier).fetchNearby(
                    lat: event.camera.center.latitude,
                    lng: event.camera.center.longitude);
                  ref.read(bountiesProvider.notifier).fetchNearby(
                    lat: event.camera.center.latitude,
                    lng: event.camera.center.longitude);
                }
              },
            ),
            children: [
              TileLayer(
                urlTemplate: 'https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}',
                userAgentPackageName: 'com.laesulia.app',
              ),
              MarkerLayer(
                markers: locState.locations.map((loc) => Marker(
                  point: loc.latLng, width: 120, height: 60,
                  child: GestureDetector(
                    onTap: () => _showLocationInfo(loc),
                    child: Column(children: [
                      Container(
                        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                        decoration: BoxDecoration(
                          color: const Color(0xFF0066CC),
                          borderRadius: BorderRadius.circular(8),
                          boxShadow: [BoxShadow(color: Colors.black.withOpacity(0.2), blurRadius: 4)],
                        ),
                        child: Text('${loc.emoji} ${loc.displayName}',
                          style: GoogleFonts.outfit(color: Colors.white, fontSize: 10, fontWeight: FontWeight.w600),
                          overflow: TextOverflow.ellipsis),
                      ),
                      const Icon(Icons.location_on, color: Color(0xFF0066CC), size: 20),
                    ]),
                  ),
                )).toList(),
              ),
              MarkerLayer(
                markers: bountyState.bounties.map((bounty) {
                  final color = bounty.status == 'open' ? Colors.red
                    : bounty.status == 'claimed' ? Colors.orange : Colors.green;
                  return Marker(
                    point: bounty.latLng, width: 140, height: 70,
                    child: GestureDetector(
                      onTap: () => _showBountyInfo(bounty),
                      child: Column(children: [
                        Container(
                          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                          decoration: BoxDecoration(
                            color: color,
                            borderRadius: BorderRadius.circular(8),
                            boxShadow: [BoxShadow(color: Colors.black.withOpacity(0.3), blurRadius: 4)],
                          ),
                          child: Text('${bounty.statusEmoji} \$${bounty.rewardSbd.toStringAsFixed(0)} SBD',
                            style: GoogleFonts.outfit(color: Colors.white, fontSize: 10, fontWeight: FontWeight.w700)),
                        ),
                        Icon(Icons.location_on, color: color, size: 20),
                      ]),
                    ),
                  );
                }).toList(),
              ),
            ],
          ),

          // Top bar
          SafeArea(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Container(
                padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
                decoration: BoxDecoration(
                  color: Colors.white,
                  borderRadius: BorderRadius.circular(12),
                  boxShadow: [BoxShadow(color: Colors.black.withOpacity(0.1), blurRadius: 8)],
                ),
                child: Row(children: [
                  const Icon(Icons.map, color: Color(0xFF0066CC), size: 20),
                  const SizedBox(width: 8),
                  Text('Laesulia', style: GoogleFonts.fraunces(fontSize: 18, fontWeight: FontWeight.w600, color: const Color(0xFF0A1628))),
                  const SizedBox(width: 4),
                  Text('· Worldwide', style: GoogleFonts.outfit(color: Colors.grey[500], fontSize: 13)),
                  const Spacer(),
                  if (locState.isLoading)
                    const SizedBox(width: 14, height: 14, child: CircularProgressIndicator(strokeWidth: 2))
                  else
                    Text('${locState.locations.length} places', style: GoogleFonts.outfit(color: Colors.grey[500], fontSize: 12)),
                  const SizedBox(width: 8),
                  GestureDetector(
                    onTap: () async {
                      await ref.read(authProvider.notifier).logout();
                      if (mounted) context.go('/login');
                    },
                    child: const Icon(Icons.logout, color: Colors.grey, size: 18),
                  ),
                ]),
              ),
            ),
          ),

          // Bounty drop mode banner
          if (_bountyDropMode)
            Positioned(
              top: 90, left: 0, right: 0,
              child: Center(
                child: Container(
                  padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 10),
                  decoration: BoxDecoration(
                    color: Colors.red,
                    borderRadius: BorderRadius.circular(24),
                    boxShadow: [BoxShadow(color: Colors.red.withOpacity(0.4), blurRadius: 12)],
                  ),
                  child: Row(mainAxisSize: MainAxisSize.min, children: [
                    const Icon(Icons.touch_app, color: Colors.white, size: 18),
                    const SizedBox(width: 8),
                    Text('Tap map to drop bounty pin',
                      style: GoogleFonts.outfit(color: Colors.white, fontWeight: FontWeight.w600)),
                    const SizedBox(width: 12),
                    GestureDetector(
                      onTap: () => setState(() => _bountyDropMode = false),
                      child: const Icon(Icons.close, color: Colors.white, size: 18)),
                  ]),
                ),
              ),
            ),
        ],
      ),

      floatingActionButton: Column(
        mainAxisAlignment: MainAxisAlignment.end,
        children: [
          FloatingActionButton(
            heroTag: "add",
            backgroundColor: const Color(0xFF0066CC),
            onPressed: () => _showAddLocation(context),
            child: const Icon(Icons.add_location_alt, color: Colors.white),
          ),
          const SizedBox(height: 8),
          FloatingActionButton(
            heroTag: "bounty",
            backgroundColor: _bountyDropMode ? Colors.red : Colors.white,
            onPressed: () => setState(() => _bountyDropMode = !_bountyDropMode),
            child: Icon(Icons.monetization_on,
              color: _bountyDropMode ? Colors.white : Colors.red),
          ),
          const SizedBox(height: 8),
          FloatingActionButton.small(
            heroTag: "zoom_in",
            backgroundColor: Colors.white,
            onPressed: () => _mapController.move(_mapController.camera.center, _mapController.camera.zoom + 1),
            child: const Icon(Icons.add, color: Color(0xFF0066CC)),
          ),
          const SizedBox(height: 4),
          FloatingActionButton.small(
            heroTag: "zoom_out",
            backgroundColor: Colors.white,
            onPressed: () => _mapController.move(_mapController.camera.center, _mapController.camera.zoom - 1),
            child: const Icon(Icons.remove, color: Color(0xFF0066CC)),
          ),
        ],
      ),
    );
  }

  void _showDropBountySheet(double lat, double lng) {
    final titleCtrl = TextEditingController();
    final descCtrl  = TextEditingController();
    double reward   = 5.0;

    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.transparent,
      builder: (_) => StatefulBuilder(
        builder: (context, setModalState) => Container(
          decoration: const BoxDecoration(
            color: Colors.white,
            borderRadius: BorderRadius.vertical(top: Radius.circular(20))),
          padding: EdgeInsets.only(
            top: 16, left: 20, right: 20,
            bottom: MediaQuery.of(context).viewInsets.bottom + 24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Center(child: Container(width: 40, height: 4,
                decoration: BoxDecoration(color: Colors.grey[300], borderRadius: BorderRadius.circular(2)))),
              const SizedBox(height: 16),
              Text('🎯 Drop Bounty Pin',
                style: GoogleFonts.fraunces(fontSize: 20, fontWeight: FontWeight.w600)),
              Text('${lat.toStringAsFixed(5)}, ${lng.toStringAsFixed(5)}',
                style: GoogleFonts.outfit(color: Colors.grey[500], fontSize: 12)),
              const SizedBox(height: 16),
              TextField(
                controller: titleCtrl,
                decoration: InputDecoration(
                  labelText: 'Title *',
                  border: OutlineInputBorder(borderRadius: BorderRadius.circular(10)),
                  focusedBorder: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(10),
                    borderSide: const BorderSide(color: Colors.red, width: 2)),
                ),
                textCapitalization: TextCapitalization.words,
              ),
              const SizedBox(height: 12),
              TextField(
                controller: descCtrl,
                decoration: InputDecoration(
                  labelText: 'Instructions for field worker',
                  border: OutlineInputBorder(borderRadius: BorderRadius.circular(10))),
              ),
              const SizedBox(height: 12),
              Row(children: [
                Text('Reward:', style: GoogleFonts.outfit(fontWeight: FontWeight.w600)),
                const SizedBox(width: 12),
                DropdownButton<double>(
                  value: reward,
                  items: [5.0, 8.0, 10.0, 15.0].map((v) =>
                    DropdownMenuItem(value: v, child: Text('\$$v SBD', style: GoogleFonts.outfit()))).toList(),
                  onChanged: (v) => setModalState(() => reward = v!),
                ),
              ]),
              const SizedBox(height: 20),
              SizedBox(
                width: double.infinity,
                child: ElevatedButton(
                  style: ElevatedButton.styleFrom(
                    backgroundColor: Colors.red,
                    padding: const EdgeInsets.symmetric(vertical: 14)),
                  onPressed: () async {
                    if (titleCtrl.text.trim().isEmpty) return;
                    Navigator.pop(context);
                    await _createBounty(
                      title: titleCtrl.text.trim(),
                      description: descCtrl.text.trim(),
                      lat: lat, lng: lng, reward: reward);
                  },
                  child: Text('🎯 Drop Bounty Pin',
                    style: GoogleFonts.outfit(color: Colors.white, fontWeight: FontWeight.w700, fontSize: 15)),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _createBounty({
    required String title,
    required String description,
    required double lat,
    required double lng,
    required double reward,
  }) async {
    final token = ref.read(authProvider).token;
    if (token == null) return;
    try {
      final response = await http.post(
        Uri.parse('$_apiBase/api/v1/bounties'),
        headers: {'Content-Type': 'application/json', 'Authorization': 'Bearer $token'},
        body: jsonEncode({'title': title, 'description': description,
          'lat': lat, 'lng': lng, 'reward_sbd': reward, 'submit_type': 'both'}),
      );
      if (response.statusCode == 201) {
        await ref.read(bountiesProvider.notifier).fetchNearby(lat: lat, lng: lng);
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(SnackBar(
            content: Text('🎯 Bounty pin dropped!', style: GoogleFonts.outfit()),
            backgroundColor: Colors.red,
            behavior: SnackBarBehavior.floating,
          ));
        }
      }
    } catch (e) {
      debugPrint('Error: $e');
    }
  }

  void _showLocationInfo(WanLocation loc) {
    showModalBottomSheet(
      context: context,
      backgroundColor: Colors.transparent,
      builder: (_) => Container(
        decoration: const BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.vertical(top: Radius.circular(20))),
        padding: const EdgeInsets.all(24),
        child: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            Text(loc.emoji, style: const TextStyle(fontSize: 28)),
            const SizedBox(width: 12),
            Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Text(loc.name, style: GoogleFonts.fraunces(fontSize: 20, fontWeight: FontWeight.w600)),
              if (loc.localName.isNotEmpty)
                Text(loc.localName, style: GoogleFonts.outfit(color: Colors.grey[600], fontSize: 14)),
            ])),
            if (loc.verified)
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                decoration: BoxDecoration(color: Colors.green[50], borderRadius: BorderRadius.circular(8)),
                child: Text('✅ Verified', style: GoogleFonts.outfit(color: Colors.green[700], fontSize: 12))),
          ]),
          const SizedBox(height: 16),
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(color: Colors.grey[50], borderRadius: BorderRadius.circular(8)),
            child: Row(children: [
              const Icon(Icons.tag, size: 16, color: Colors.grey),
              const SizedBox(width: 8),
              Text(loc.addressCode, style: GoogleFonts.outfit(fontWeight: FontWeight.w700, fontSize: 16, color: const Color(0xFF0066CC))),
              const Spacer(),
              Text('${loc.upvotes} ▲', style: GoogleFonts.outfit(color: Colors.grey)),
            ]),
          ),
        ]),
      ),
    );
  }

  void _showBountyInfo(Bounty bounty) {
    showModalBottomSheet(
      context: context,
      backgroundColor: Colors.transparent,
      builder: (_) => Container(
        decoration: const BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.vertical(top: Radius.circular(20))),
        padding: const EdgeInsets.all(24),
        child: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            Text(bounty.statusEmoji, style: const TextStyle(fontSize: 28)),
            const SizedBox(width: 12),
            Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Text(bounty.title, style: GoogleFonts.fraunces(fontSize: 20, fontWeight: FontWeight.w600)),
              Text('Submit: ${bounty.submitType}', style: GoogleFonts.outfit(color: Colors.grey[600], fontSize: 13)),
            ])),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
              decoration: BoxDecoration(
                color: Colors.red[50], borderRadius: BorderRadius.circular(12),
                border: Border.all(color: Colors.red[200]!)),
              child: Text('\$${bounty.rewardSbd.toStringAsFixed(0)} SBD',
                style: GoogleFonts.outfit(color: Colors.red[700], fontWeight: FontWeight.w700, fontSize: 16)),
            ),
          ]),
          if (bounty.description.isNotEmpty) ...[
            const SizedBox(height: 12),
            Text(bounty.description, style: GoogleFonts.outfit(color: Colors.grey[700])),
          ],
          const SizedBox(height: 20),
          if (bounty.status == 'open')
            SizedBox(
              width: double.infinity,
              child: ElevatedButton(
                style: ElevatedButton.styleFrom(
                  backgroundColor: Colors.red,
                  padding: const EdgeInsets.symmetric(vertical: 14)),
                onPressed: () async {
                  debugPrint('=== BOUNTY ID: "${bounty.id}" ===');
                  Navigator.pop(context);
                  final ok = await ref.read(bountiesProvider.notifier).claim(bounty.id);
                  if (mounted) {
                    if (ok) {
                      Navigator.push(context, MaterialPageRoute(
                        builder: (_) => UploadScreen(bounty: bounty)));
                    } else {
                      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
                        content: const Text('❌ Already claimed.'),
                        backgroundColor: Colors.red,
                        behavior: SnackBarBehavior.floating));
                    }
                  }
                },
                child: Text('🎯 Claim — \$${bounty.rewardSbd.toStringAsFixed(0)} SBD',
                  style: GoogleFonts.outfit(color: Colors.white, fontWeight: FontWeight.w700, fontSize: 15)),
              ),
            ),
        ]),
      ),
    );
  }

  void _showAddLocation(BuildContext context) {
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(
      content: Text('📍 Add location feature coming next!', style: GoogleFonts.outfit()),
      backgroundColor: const Color(0xFF0066CC),
      behavior: SnackBarBehavior.floating,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
    ));
  }
}
