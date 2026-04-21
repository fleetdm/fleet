#!/bin/bash
# Deletes orphaned duplicate certificates matching a given CN from the login
# keychain, keeping the most recently issued one (the one tied to the current
# profile).
#
# Usage: ./delete-scep-certs.sh [-y] [-a] [-u username] <certificate common name>
#   -y  Skip confirmation prompt
#   -a  Remove all matching certificates (including the newest)
#   -u  Target a specific user's login keychain (required when running as root)
#
# Example: ./delete-scep-certs.sh "Fleet conditional access for Okta"

set -e

auto_confirm=false
remove_all=false
target_user=""
while getopts "yau:" opt; do
    case "$opt" in
        y) auto_confirm=true ;;
        a) remove_all=true ;;
        u) target_user="$OPTARG" ;;
        *)
            echo "Usage: $0 [-y] [-a] [-u username] <certificate common name>" >&2
            exit 1
            ;;
    esac
done
shift $((OPTIND - 1))

if [ $# -eq 0 ]; then
    echo "Usage: $0 [-y] [-a] [-u username] <certificate common name>" >&2
    echo "  -y  Skip confirmation prompt" >&2
    echo "  -a  Remove all matching certificates (including the newest)" >&2
    echo "  -u  Target a specific user's login keychain (required when running as root)" >&2
    exit 1
fi

CN="$1"

# Resolve the keychain path.
if [ -n "$target_user" ]; then
    KEYCHAIN="/Users/$target_user/Library/Keychains/login.keychain-db"
    if [ ! -f "$KEYCHAIN" ]; then
        echo "Error: keychain not found at $KEYCHAIN" >&2
        exit 1
    fi
elif [ "$(id -u)" -eq 0 ]; then
    echo "Error: running as root without -u flag. Specify the target user with -u <username>." >&2
    exit 1
else
    KEYCHAIN="login.keychain-db"
fi

# Collect SHA-1 hash and Not Before date for every matching certificate.
# Output is written to a temp file as: <epoch> <hash>
tmpfile=$(mktemp)
trap 'rm -f "$tmpfile" "$tmpfile.raw" "$tmpfile.err"' EXIT

security find-certificate -a -c "$CN" -Z -p "$KEYCHAIN" >"$tmpfile.raw" 2>"$tmpfile.err" || true

# Split the raw output into individual cert blocks and extract hash + date.
current_hash=""
current_pem=""
while IFS= read -r line; do
    case "$line" in
        "SHA-1 hash:"*)
            current_hash=$(echo "$line" | awk '{print $NF}')
            ;;
        "-----BEGIN CERTIFICATE-----")
            current_pem="$line"$'\n'
            ;;
        "-----END CERTIFICATE-----")
            current_pem+="$line"$'\n'
            not_before=$(echo "$current_pem" | openssl x509 -noout -startdate 2>/dev/null | cut -d= -f2)
            epoch=$(date -j -f "%b %e %T %Y %Z" "$not_before" "+%s" 2>/dev/null || echo "0")
            echo "$epoch $current_hash" >> "$tmpfile"
            current_pem=""
            ;;
        *)
            if [ -n "$current_pem" ]; then
                current_pem+="$line"$'\n'
            fi
            ;;
    esac
done < "$tmpfile.raw"

total=$(wc -l < "$tmpfile" | tr -d ' ')

if [ "$total" -eq 0 ]; then
    echo "No certificates found matching \"$CN\""
    exit 0
fi

if [ "$total" -eq 1 ] && [ "$remove_all" = false ]; then
    echo "Only one certificate found matching \"$CN\", nothing to delete."
    exit 0
fi

# Sort by epoch descending; the first line is the newest.
newest_hash=$(sort -rn "$tmpfile" | head -1 | awk '{print $2}')

if [ "$remove_all" = true ]; then
    echo "Found $total certificate(s) matching \"$CN\""
    echo "  Will delete:     ALL $total certificate(s):"
    while read -r epoch hash; do
        issued=$(date -r "$epoch" "+%Y-%m-%d %H:%M:%S" 2>/dev/null || echo "unknown")
        echo "    $hash (issued $issued)"
    done < "$tmpfile"
else
    to_delete=$((total - 1))
    echo "Found $total certificate(s) matching \"$CN\""
    echo "  Keeping newest:  $newest_hash"
    echo "  Will delete:     $to_delete orphaned certificate(s):"
    while read -r epoch hash; do
        if [ "$hash" = "$newest_hash" ]; then
            continue
        fi
        issued=$(date -r "$epoch" "+%Y-%m-%d %H:%M:%S" 2>/dev/null || echo "unknown")
        echo "    $hash (issued $issued)"
    done < "$tmpfile"
fi

if [ "$auto_confirm" = false ]; then
    printf "\nProceed? [y/N] "
    read -r answer
    if [ "$answer" != "y" ] && [ "$answer" != "Y" ]; then
        echo "Aborted."
        exit 0
    fi
fi

deleted=0
while read -r epoch hash; do
    if [ "$remove_all" = false ] && [ "$hash" = "$newest_hash" ]; then
        continue
    fi
    echo "Deleting $hash"
    security delete-identity -Z "$hash" "$KEYCHAIN"
    deleted=$((deleted + 1))
done < "$tmpfile"

if [ "$remove_all" = true ]; then
    echo "Done. Deleted all $deleted certificate(s)."
else
    echo "Done. Deleted $deleted orphaned certificate(s), kept 1."
fi
