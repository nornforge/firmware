#!/usr/bin/env bash
# Checks on a connected board that the firmware stays in sync with the host
# after stray bytes or a frame that was cut off. Each case writes bad input to
# the serial port, waits, and expects `norn version` to answer right away.
#
# Usage: sudo scripts/check-serial-recovery.sh [port]   (default /dev/ttyACM0)
set -euo pipefail

PORT="${1:-/dev/ttyACM0}"
# Longer than the firmware's timeout for an incomplete frame (100 ms).
PAUSE=0.3

stty -F "$PORT" raw -echo

failed=0
check() {
    local name="$1" input="$2"
    printf '%s' "$input" > "$PORT"
    sleep "$PAUSE"
    if norn --device "$PORT" version > /dev/null 2>&1; then
        echo "ok    $name"
    else
        echo "FAIL  $name"
        failed=1
        # Get back in sync for the next case.
        for _ in 1 2 3 4 5; do norn --device "$PORT" version > /dev/null 2>&1 && break; done
    fi
}

norn --device "$PORT" version > /dev/null || { echo "the board does not answer on $PORT"; exit 2; }

# ModemManager probes new ttyACM devices with this kind of input.
check "stray bytes (AT probe)" $'AT\r'
# A frame cut off within its length field.
check "frame cut off in the length field" 'E979#0000000'
# A frame cut off within its payload.
check "frame cut off in the payload" 'E979#000000037#7a3c9f10{"channel":0,'

exit "$failed"
