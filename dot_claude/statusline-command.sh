#!/usr/bin/env bash
# Claude Code statusline: [Model - effort] ctx % | 5h % - time left | 7d %

# Catppuccin Mocha
BLUE=$'\e[38;2;137;180;250m'    # model name
MAUVE=$'\e[38;2;203;166;247m'   # effort level
OVERLAY=$'\e[38;2;108;112;134m' # brackets, separators
SUBTEXT=$'\e[38;2;166;173;200m' # labels
GREEN=$'\e[38;2;166;227;161m'   # < 50%
PEACH=$'\e[38;2;250;179;135m'   # 50-79%
RED=$'\e[38;2;243;139;168m'     # >= 80%
TEAL=$'\e[38;2;148;226;213m'    # time remaining
BOLD=$'\e[1m'
RESET=$'\e[0m'

pct_color() {
  local pct=$1
  if   (( pct >= 80 )); then printf '%s' "$RED"
  elif (( pct >= 50 )); then printf '%s' "$PEACH"
  else                       printf '%s' "$GREEN"
  fi
}

json=$(cat)

MODEL=$(printf '%s' "$json"    | jq -r '.model.display_name // "unknown"' | sed 's/^[Cc]laude //')
EFFORT=$(printf '%s' "$json"   | jq -r '.effort.level // ""')
CTX_PCT=$(printf '%s' "$json"  | jq -r '.context_window.used_percentage // 0 | floor')
FIVE_PCT=$(printf '%s' "$json" | jq -r '.rate_limits.five_hour.used_percentage // 0 | floor')
SEVEN_PCT=$(printf '%s' "$json"| jq -r '.rate_limits.seven_day.used_percentage // 0 | floor')
RESETS_AT=$(printf '%s' "$json"| jq -r '.rate_limits.five_hour.resets_at // ""')

# Time until 5h window resets
time_left=""
if [[ -n "$RESETS_AT" && "$RESETS_AT" != "null" ]]; then
  now=$(date +%s)
  diff=$(( RESETS_AT - now ))
  if   (( diff > 3600 )); then
    time_left="$(( diff / 3600 ))h$(( (diff % 3600) / 60 ))m"
  elif (( diff > 0 )); then
    time_left="$(( diff / 60 ))m"
  else
    time_left="now"
  fi
fi

# [Model - effort]
model_block="${OVERLAY}[${BLUE}${BOLD}${MODEL}${RESET}"
[[ -n "$EFFORT" ]] && model_block+="${OVERLAY} - ${MAUVE}${EFFORT}${RESET}"
model_block+="${OVERLAY}]${RESET}"

# ctx X%
ctx_block="${SUBTEXT}ctx${RESET} $(pct_color $CTX_PCT)${CTX_PCT}%${RESET}"

# 5h X% - Xhm
five_block="${SUBTEXT}5h${RESET} $(pct_color $FIVE_PCT)${FIVE_PCT}%${RESET}"
[[ -n "$time_left" ]] && five_block+="${OVERLAY} - ${TEAL}${time_left}${RESET}"

# 7d X%
seven_block="${SUBTEXT}7d${RESET} $(pct_color $SEVEN_PCT)${SEVEN_PCT}%${RESET}"

sep="${OVERLAY} | ${RESET}"

printf '%s %s%s%s%s%s\n' \
  "$model_block" \
  "$ctx_block" "$sep" \
  "$five_block" "$sep" \
  "$seven_block"
