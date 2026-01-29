from __future__ import annotations

import os


def get_rss_bytes() -> int:
    try:
        import resource

        rss = resource.getrusage(resource.RUSAGE_SELF).ru_maxrss
        # Heuristic: macOS reports bytes, Linux reports kilobytes.
        if rss < 10 * 1024 * 1024:
            return int(rss * 1024)
        return int(rss)
    except Exception:
        return 0


def get_pid() -> int:
    return os.getpid()
