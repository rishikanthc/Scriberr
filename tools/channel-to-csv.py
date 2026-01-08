#!/usr/bin/env python3
"""
YouTube Channel to CSV Exporter

Extracts all videos from a YouTube channel and exports metadata to CSV.
Designed to work with Scriberr's CSV batch processor.

Usage:
    python channel-to-csv.py <channel_url> [options]

Examples:
    python channel-to-csv.py https://www.youtube.com/@ChannelName
    python channel-to-csv.py https://www.youtube.com/channel/UCxxxxxx --output videos.csv
    python channel-to-csv.py https://www.youtube.com/c/ChannelName --limit 100
"""

import argparse
import csv
import json
import subprocess
import sys
from datetime import datetime
from pathlib import Path


def format_duration(seconds):
    """Convert seconds to HH:MM:SS or MM:SS format."""
    if seconds is None:
        return "N/A"
    hours, remainder = divmod(int(seconds), 3600)
    minutes, secs = divmod(remainder, 60)
    if hours > 0:
        return f"{hours}:{minutes:02d}:{secs:02d}"
    return f"{minutes}:{secs:02d}"


def format_views(view_count):
    """Format view count with commas."""
    if view_count is None:
        return "N/A"
    return f"{view_count:,}"


def format_date(date_str):
    """Format upload date from YYYYMMDD to YYYY-MM-DD."""
    if not date_str or len(date_str) != 8:
        return date_str or "N/A"
    try:
        return f"{date_str[:4]}-{date_str[4:6]}-{date_str[6:]}"
    except Exception:
        return date_str


def get_channel_videos(channel_url, limit=None, verbose=False):
    """
    Fetch video metadata from a YouTube channel using yt-dlp.

    Args:
        channel_url: URL of the YouTube channel
        limit: Maximum number of videos to fetch (None for all)
        verbose: Print progress information

    Returns:
        List of video metadata dictionaries
    """
    cmd = [
        "yt-dlp",
        "--flat-playlist",
        "--dump-json",
        "--no-warnings",
        "--ignore-errors",
        channel_url
    ]

    if limit:
        cmd.extend(["--playlist-end", str(limit)])

    if verbose:
        print(f"Fetching video list from: {channel_url}")
        if limit:
            print(f"Limiting to {limit} videos")

    try:
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=300  # 5 minute timeout
        )
    except subprocess.TimeoutExpired:
        print("Error: Request timed out. The channel may have too many videos.", file=sys.stderr)
        sys.exit(1)
    except FileNotFoundError:
        print("Error: yt-dlp not found. Please install it: pip install yt-dlp", file=sys.stderr)
        sys.exit(1)

    if result.returncode != 0 and not result.stdout:
        print(f"Error fetching channel: {result.stderr}", file=sys.stderr)
        sys.exit(1)

    videos = []
    for line in result.stdout.strip().split('\n'):
        if line:
            try:
                video = json.loads(line)
                videos.append(video)
            except json.JSONDecodeError:
                continue

    return videos


def get_video_details(video_ids, verbose=False):
    """
    Fetch detailed metadata for a list of video IDs.

    Args:
        video_ids: List of YouTube video IDs
        verbose: Print progress information

    Returns:
        Dictionary mapping video ID to metadata
    """
    if not video_ids:
        return {}

    details = {}
    total = len(video_ids)

    for i, video_id in enumerate(video_ids, 1):
        if verbose:
            print(f"\rFetching details: {i}/{total}", end="", flush=True)

        url = f"https://www.youtube.com/watch?v={video_id}"
        cmd = [
            "yt-dlp",
            "--dump-json",
            "--no-warnings",
            "--no-download",
            url
        ]

        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=30
            )
            if result.returncode == 0 and result.stdout:
                details[video_id] = json.loads(result.stdout)
        except (subprocess.TimeoutExpired, json.JSONDecodeError):
            continue

    if verbose:
        print()  # New line after progress

    return details


def export_to_csv(videos, output_path, fetch_details=False, verbose=False):
    """
    Export video metadata to CSV file.

    Args:
        videos: List of video metadata from flat playlist
        output_path: Path to output CSV file
        fetch_details: Whether to fetch full details (slower but more accurate)
        verbose: Print progress information
    """
    if not videos:
        print("No videos found.", file=sys.stderr)
        return 0

    # Get detailed info if requested
    details = {}
    if fetch_details:
        video_ids = [v.get('id') for v in videos if v.get('id')]
        details = get_video_details(video_ids, verbose)

    fieldnames = ['url', 'title', 'duration', 'views', 'upload_date', 'video_id']

    with open(output_path, 'w', newline='', encoding='utf-8') as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()

        for video in videos:
            video_id = video.get('id', '')

            # Use detailed info if available
            if video_id in details:
                detail = details[video_id]
                duration = detail.get('duration')
                view_count = detail.get('view_count')
                upload_date = detail.get('upload_date')
            else:
                duration = video.get('duration')
                view_count = video.get('view_count')
                upload_date = video.get('upload_date')

            row = {
                'url': f"https://www.youtube.com/watch?v={video_id}" if video_id else video.get('url', ''),
                'title': video.get('title', 'Unknown'),
                'duration': format_duration(duration),
                'views': format_views(view_count),
                'upload_date': format_date(upload_date),
                'video_id': video_id
            }
            writer.writerow(row)

    return len(videos)


def main():
    parser = argparse.ArgumentParser(
        description='Extract YouTube channel videos to CSV for batch transcription.',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog='''
Examples:
  %(prog)s https://www.youtube.com/@ChannelName
  %(prog)s https://www.youtube.com/channel/UCxxxxxx -o videos.csv
  %(prog)s https://www.youtube.com/c/ChannelName --limit 50 --details

Output CSV columns:
  url          - Full YouTube video URL
  title        - Video title
  duration     - Video length (HH:MM:SS or MM:SS)
  views        - View count
  upload_date  - Upload date (YYYY-MM-DD)
  video_id     - YouTube video ID

The output CSV is compatible with Scriberr's CSV batch processor.
        '''
    )

    parser.add_argument(
        'channel_url',
        help='YouTube channel URL (supports @handle, /channel/, /c/, /user/ formats)'
    )
    parser.add_argument(
        '-o', '--output',
        default=None,
        help='Output CSV file path (default: channel_videos_YYYYMMDD.csv)'
    )
    parser.add_argument(
        '-l', '--limit',
        type=int,
        default=None,
        help='Maximum number of videos to fetch (default: all)'
    )
    parser.add_argument(
        '-d', '--details',
        action='store_true',
        help='Fetch full video details (slower but more accurate view counts)'
    )
    parser.add_argument(
        '-v', '--verbose',
        action='store_true',
        help='Show progress information'
    )
    parser.add_argument(
        '--version',
        action='version',
        version='%(prog)s 1.0.0'
    )

    args = parser.parse_args()

    # Validate channel URL
    valid_patterns = ['youtube.com/', 'youtu.be/']
    if not any(p in args.channel_url for p in valid_patterns):
        print("Error: Invalid YouTube URL. Please provide a valid channel URL.", file=sys.stderr)
        print("Examples:", file=sys.stderr)
        print("  https://www.youtube.com/@ChannelName", file=sys.stderr)
        print("  https://www.youtube.com/channel/UCxxxxxx", file=sys.stderr)
        print("  https://www.youtube.com/c/ChannelName", file=sys.stderr)
        sys.exit(1)

    # Generate default output filename
    if args.output is None:
        timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')
        args.output = f"channel_videos_{timestamp}.csv"

    # Ensure output directory exists
    output_path = Path(args.output)
    output_path.parent.mkdir(parents=True, exist_ok=True)

    if args.verbose:
        print("=" * 60)
        print("YouTube Channel to CSV Exporter")
        print("=" * 60)

    # Fetch videos
    videos = get_channel_videos(args.channel_url, args.limit, args.verbose)

    if not videos:
        print("No videos found on this channel.", file=sys.stderr)
        sys.exit(1)

    if args.verbose:
        print(f"Found {len(videos)} videos")

    # Export to CSV
    count = export_to_csv(videos, args.output, args.details, args.verbose)

    print(f"Exported {count} videos to: {args.output}")

    # Print summary
    if args.verbose:
        print("\nCSV columns: url, title, duration, views, upload_date, video_id")
        print(f"\nTo process with Scriberr:")
        print(f"  ./scripts/csvbatch.sh --csv {args.output}")


if __name__ == '__main__':
    main()
