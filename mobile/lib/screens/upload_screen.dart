
import 'dart:typed_data';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:http/http.dart' as http;
import 'package:image_picker/image_picker.dart';
import '../providers/auth_provider.dart';
import '../providers/bounties_provider.dart';


const _apiBase = 'https://laesulia-api.onrender.com';

class UploadScreen extends ConsumerStatefulWidget {
  final Bounty bounty;
  const UploadScreen({super.key, required this.bounty});

  @override
  ConsumerState<UploadScreen> createState() => _UploadScreenState();
}

class _UploadScreenState extends ConsumerState<UploadScreen> {
  final List<Uint8List> _files     = [];
  final List<String>    _fileNames = [];
  bool _isUploading  = false;
  bool _isSubmitting = false;
  String? _message;

  int get _required => 3;

  @override
  Widget build(BuildContext context) {
    final progress = _files.isEmpty ? 0.0 : _files.length / _required;

    return Scaffold(
      appBar: AppBar(
        title: Text('Upload Files', style: GoogleFonts.fraunces(fontSize: 20)),
        backgroundColor: Colors.white,
        elevation: 0,
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Container(
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(
                color: Colors.red[50],
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: Colors.red[200]!),
              ),
              child: Row(children: [
                const Text('🎯', style: TextStyle(fontSize: 28)),
                const SizedBox(width: 12),
                Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                  Text(widget.bounty.title,
                    style: GoogleFonts.fraunces(fontSize: 16, fontWeight: FontWeight.w600)),
                  Text(widget.bounty.description,
                    style: GoogleFonts.outfit(color: Colors.grey[700], fontSize: 13)),
                ])),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                  decoration: BoxDecoration(color: Colors.red, borderRadius: BorderRadius.circular(8)),
                  child: Text('\$${widget.bounty.rewardSbd.toStringAsFixed(0)} SBD',
                    style: GoogleFonts.outfit(color: Colors.white, fontWeight: FontWeight.w700)),
                ),
              ]),
            ),
            const SizedBox(height: 24),

            Text('Progress', style: GoogleFonts.outfit(fontWeight: FontWeight.w700, fontSize: 14)),
            const SizedBox(height: 8),
            ClipRRect(
              borderRadius: BorderRadius.circular(8),
              child: LinearProgressIndicator(
                value: progress,
                minHeight: 10,
                backgroundColor: Colors.grey[200],
                valueColor: AlwaysStoppedAnimation<Color>(
                  progress >= 1 ? Colors.green : Colors.red),
              ),
            ),
            const SizedBox(height: 6),
            Text('${_files.length} of $_required files uploaded',
              style: GoogleFonts.outfit(color: Colors.grey[600], fontSize: 13)),
            const SizedBox(height: 24),

            Text('Instructions', style: GoogleFonts.outfit(fontWeight: FontWeight.w700, fontSize: 14)),
            const SizedBox(height: 8),
            Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(color: Colors.blue[50], borderRadius: BorderRadius.circular(8)),
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                _instruction('📍', 'Go to the exact GPS location on the map'),
                _instruction('📸', 'Take $_required clear photos/videos'),
                _instruction('☀️', 'Make sure lighting is good'),
                _instruction('🔄', 'Show different angles of the location'),
              ]),
            ),
            const SizedBox(height: 24),

            if (_files.isNotEmpty) ...[
              Text('Uploaded Files', style: GoogleFonts.outfit(fontWeight: FontWeight.w700, fontSize: 14)),
              const SizedBox(height: 8),
              GridView.builder(
                shrinkWrap: true,
                physics: const NeverScrollableScrollPhysics(),
                gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
                  crossAxisCount: 3, crossAxisSpacing: 8, mainAxisSpacing: 8),
                itemCount: _files.length,
                itemBuilder: (context, i) => Stack(children: [
                  ClipRRect(
                    borderRadius: BorderRadius.circular(8),
                    child: Image.memory(_files[i], fit: BoxFit.cover,
                      width: double.infinity, height: double.infinity)),
                  Positioned(top: 4, right: 4,
                    child: Container(
                      padding: const EdgeInsets.all(2),
                      decoration: const BoxDecoration(color: Colors.green, shape: BoxShape.circle),
                      child: const Icon(Icons.check, color: Colors.white, size: 12))),
                ]),
              ),
              const SizedBox(height: 24),
            ],

            if (_files.length < _required)
              SizedBox(
                width: double.infinity,
                child: OutlinedButton.icon(
                  onPressed: _isUploading ? null : _pickFile,
                  icon: _isUploading
                    ? const SizedBox(width: 16, height: 16,
                        child: CircularProgressIndicator(strokeWidth: 2))
                    : const Icon(Icons.add_a_photo),
                  label: Text(
                    _isUploading ? 'Uploading...'
                      : 'Add Photo / Video (${_files.length}/$_required)',
                    style: GoogleFonts.outfit(fontWeight: FontWeight.w600)),
                  style: OutlinedButton.styleFrom(
                    foregroundColor: Colors.red,
                    side: const BorderSide(color: Colors.red),
                    padding: const EdgeInsets.symmetric(vertical: 14)),
                ),
              ),

            if (_message != null) ...[
              const SizedBox(height: 12),
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: _message!.startsWith('✅') ? Colors.green[50] : Colors.red[50],
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(
                    color: _message!.startsWith('✅') ? Colors.green[200]! : Colors.red[200]!)),
                child: Text(_message!,
                  style: GoogleFonts.outfit(
                    color: _message!.startsWith('✅') ? Colors.green[800] : Colors.red[800])),
              ),
            ],

            const SizedBox(height: 16),

            if (_files.length >= _required)
              SizedBox(
                width: double.infinity,
                child: ElevatedButton(
                  onPressed: _isSubmitting ? null : _submitJob,
                  style: ElevatedButton.styleFrom(
                    backgroundColor: Colors.green,
                    padding: const EdgeInsets.symmetric(vertical: 16)),
                  child: _isSubmitting
                    ? const CircularProgressIndicator(color: Colors.white)
                    : Text(
                        '✅ Submit for Review — \$${widget.bounty.rewardSbd.toStringAsFixed(0)} SBD',
                        style: GoogleFonts.outfit(
                          color: Colors.white, fontWeight: FontWeight.w700, fontSize: 15)),
                ),
              ),
          ],
        ),
      ),
    );
  }

  Widget _instruction(String emoji, String text) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 6),
      child: Row(children: [
        Text(emoji, style: const TextStyle(fontSize: 16)),
        const SizedBox(width: 8),
        Text(text, style: GoogleFonts.outfit(fontSize: 13, color: Colors.blue[900])),
      ]),
    );
  }

  Future<void> _pickFile() async {
    final picker = ImagePicker();
    final picked = await picker.pickImage(source: ImageSource.gallery, imageQuality: 80);
    if (picked == null) return;

    setState(() { _isUploading = true; _message = null; });

    try {
      final bytes   = await picked.readAsBytes();
      final token   = ref.read(authProvider).token;
      debugPrint('UPLOAD URL: $_apiBase/api/v1/bounties/${widget.bounty.id}/upload');
      
      final request = http.MultipartRequest(
        'POST',
        Uri.parse('$_apiBase/api/v1/bounties/${widget.bounty.id}/upload'),
      );
      request.headers['Authorization'] = 'Bearer $token';
      request.files.add(http.MultipartFile.fromBytes('file', bytes, filename: picked.name));

      final response = await request.send();
      final body     = await response.stream.bytesToString();

      if (response.statusCode == 200) {
        setState(() {
          _files.add(bytes);
          _fileNames.add(picked.name);
          _message = '✅ File ${_files.length} of $_required uploaded!';
        });
      } else {
        setState(() => _message = '❌ Upload failed (${response.statusCode}): $body');
      }
    } catch (e) {
      setState(() => _message = '❌ Error: $e');
    } finally {
      setState(() => _isUploading = false);
    }
  }

  Future<void> _submitJob() async {
    setState(() => _isSubmitting = true);
    try {
      final token    = ref.read(authProvider).token;
      final response = await http.post(
        Uri.parse('$_apiBase/api/v1/bounties/${widget.bounty.id}/submit'),
        headers: {'Authorization': 'Bearer $token', 'Content-Type': 'application/json'},
        body: '{"files":[]}',
      );
      if (response.statusCode == 200) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(SnackBar(
            content: Text('🎉 Submitted! Waiting for admin review.',
              style: GoogleFonts.outfit()),
            backgroundColor: Colors.green,
            behavior: SnackBarBehavior.floating,
          ));
          Navigator.pop(context);
        }
      } else {
        setState(() => _message = '❌ Submit failed: ${response.body}');
      }
    } catch (e) {
      setState(() => _message = '❌ Submit error: $e');
    } finally {
      setState(() => _isSubmitting = false);
    }
  }
}
