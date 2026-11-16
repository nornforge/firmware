#!/usr/bin/env bash
# Checks on a connected board that the firmware stays in sync with the host
# after stray bytes or a frame that was cut off. Each case writes bad input to
# the serial port, waits for the given pause, and expects `norn version` to
# answer right away.
#
# A pause longer than the firmware's timeout for an incomplete frame (100 ms)
# lets the firmware discard the bad input before the command arrives. With a
# short or no pause the bad input and the command arrive together; the
# firmware then has to find the start of the command itself, which needs
# firmware built with norn v0.3.0 or newer.
#
# Cut-off frames are only checked with the long pause: if the command follows
# within 100 ms, the firmware takes it for the rest of the cut-off frame,
# answers that frame with an error and only then the command. The host takes
# the error as the answer, so such a case fails by design of the protocol.
#
# Usage: sudo scripts/check-serial-recovery.sh [port]   (default /dev/ttyACM0)
set -euo pipefail

PORT="${1:-/dev/ttyACM0}"

stty -F "$PORT" raw -echo

failed=0
check() {
    local name="$1" pause="$2" input="$3"
    printf '%s' "$input" > "$PORT"
    sleep "$pause"
    if norn --device "$PORT" version > /dev/null 2>&1; then
        echo "ok    $name, pause ${pause} s"
    else
        echo "FAIL  $name, pause ${pause} s"
        failed=1
        # Get back in sync for the next case. The pause lets the firmware
        # discard what is left of the bad input.
        local _
        for _ in 1 2 3 4 5; do
            sleep 0.3
            norn --device "$PORT" version > /dev/null 2>&1 && break
        done
    fi
    return 0
}

norn --device "$PORT" version > /dev/null || { echo "the board does not answer on $PORT"; exit 2; }

for pause in 0.3 0.05 0; do
    # ModemManager probes new ttyACM devices with this kind of input.
    check "stray bytes (AT probe)" "$pause" $'AT\r'
    check "repeated AT probes" "$pause" $'AT\rAT\rAT\r'
    # A marker that was started but not completed, once and twice.
    check "partial marker" "$pause" 'E97'
    check "marker started twice" "$pause" 'xxE97E979'
done

# A frame cut off within its length field.
check "frame cut off in the length field" 0.3 'E979#0000000'
# A frame cut off within its payload; the length field and its checksum are valid.
check "frame cut off in the payload" 0.3 'E979#000000037#e2da8fab{"channel":0,'

exit "$failed"
