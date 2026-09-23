#!/bin/bash

set -euo pipefail
ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR"
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color
info() { echo -e "${BLUE}ℹ️ $1${NC}"; }
success() { echo -e "${GREEN}✅ $1${NC}"; }
warning() { echo -e "${YELLOW}⚠️ $1${NC}"; }
error() { echo -e "${RED}❌ $1${NC}"; }
note() { echo -e "${CYAN}ℹ️ $1${NC}"; }
PIP_INDEX_URL="${PIP_INDEX_URL:-https://pypi.org/simple}"
GOPROXY="${GOPROXY:-https://proxy.golang.org,direct}"
ORIGINAL_PIP_INDEX_URL="${PIP_INDEX_URL:-}"
ORIGINAL_GOPROXY="${GOPROXY:-}"
show_progress() {
 local pid=$1
 local message=$2
 local i=0
 local dots=""
 if ! kill -0 "$pid" 2>/dev/null; then
 return 0
 fi
 
 while kill -0 "$pid" 2>/dev/null; do
 i=$((i + 1))
 case $((i % 4)) in
 0) dots="." ;;
 1) dots=".." ;;
 2) dots="..." ;;
 3) dots="...." ;;
 esac
 printf "\r${BLUE}⏳ %s%s${NC}" "$message" "$dots"
 sleep 0.5
 if ! kill -0 "$pid" 2>/dev/null; then
 break
 fi
 done
 printf "\r"
}

echo ""
echo "=========================================="
echo " Pyntra "
echo "=========================================="
echo ""
echo ""
warning "⚠️ Note:"
echo ""
info "Python pip :"
echo " ${PIP_INDEX_URL}"
info "Go Proxy :"
echo " ${GOPROXY}"
echo ""
note ",config"
echo ""
sleep 1

CONFIG_FILE="$ROOT_DIR/config.yaml"
VENV_DIR="$ROOT_DIR/venv"
REQUIREMENTS_FILE="$ROOT_DIR/requirements.txt"
BINARY_NAME="pyntra"
if [ ! -f "$CONFIG_FILE" ]; then
 error "config file config.yaml does not exist"
 info "directory"
 exit 1
fi
check_python() {
 if ! command -v python3 >/dev/null 2>&1; then
 error " python3"
 echo ""
 info " Python 3.10 :"
 echo " macOS: brew install python3"
 echo " Ubuntu: sudo apt-get install python3 python3-venv"
 echo " CentOS: sudo yum install python3 python3-pip"
 exit 1
 fi
 
 PYTHON_VERSION=$(python3 --version 2>&1 | awk '{print $2}')
 PYTHON_MAJOR=$(echo "$PYTHON_VERSION" | cut -d. -f1)
 PYTHON_MINOR=$(echo "$PYTHON_VERSION" | cut -d. -f2)
 
 if [ "$PYTHON_MAJOR" -lt 3 ] || ([ "$PYTHON_MAJOR" -eq 3 ] && [ "$PYTHON_MINOR" -lt 10 ]); then
 error "Python : $PYTHON_VERSION ( 3.10+)"
 exit 1
 fi
 
 success "Python : $PYTHON_VERSION"
}
check_go() {
 if ! command -v go >/dev/null 2>&1; then
 error " Go"
 echo ""
 info " Go 1.21 :"
 echo " macOS: brew install go"
 echo " Ubuntu: sudo apt-get install golang-go"
 echo " CentOS: sudo yum install golang"
 echo " : https://go.dev/dl/"
 exit 1
 fi
 
 GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
 GO_MAJOR=$(echo "$GO_VERSION" | cut -d. -f1)
 GO_MINOR=$(echo "$GO_VERSION" | cut -d. -f2)
 
 if [ "$GO_MAJOR" -lt 1 ] || ([ "$GO_MAJOR" -eq 1 ] && [ "$GO_MINOR" -lt 21 ]); then
 error "Go : $GO_VERSION ( 1.21+)"
 exit 1
 fi
 
 success "Go : $(go version)"
}
setup_python_env() {
 if [ ! -d "$VENV_DIR" ]; then
 info "create Python ..."
 python3 -m venv "$VENV_DIR"
 success "createcompleted"
 else
 info "Python "
 fi
 
 info "..."
 # shellcheck disable=SC1091
 source "$VENV_DIR/bin/activate"
 
 if [ -f "$REQUIREMENTS_FILE" ]; then
 echo ""
 note "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
 note "⚠️ pip ()"
 note " : ${PIP_INDEX_URL}"
 note " config, PIP_INDEX_URL"
 note "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
 echo ""
 
 info " pip..."
 pip install --index-url "$PIP_INDEX_URL" --upgrade pip >/dev/null 2>&1 || true
 
 info " Python ..."
 echo ""
 PIP_LOG=$(mktemp)
 (
 set +e # shelldisablederrorsign out
 pip install --index-url "$PIP_INDEX_URL" -r "$REQUIREMENTS_FILE" >"$PIP_LOG" 2>&1
 echo $? > "${PIP_LOG}.exit"
 ) &
 PIP_PID=$!
 sleep 0.1
 if kill -0 "$PIP_PID" 2>/dev/null; then
 show_progress "$PIP_PID" ""
 else
 sleep 0.2
 fi
 wait "$PIP_PID" 2>/dev/null || true
 
 PIP_EXIT_CODE=0
 if [ -f "${PIP_LOG}.exit" ]; then
 PIP_EXIT_CODE=$(cat "${PIP_LOG}.exit" 2>/dev/null || echo "1")
 rm -f "${PIP_LOG}.exit" 2>/dev/null || true
 else
 if [ -f "$PIP_LOG" ] && grep -q -i "error\|failed\|exception" "$PIP_LOG" 2>/dev/null; then
 PIP_EXIT_CODE=1
 fi
 fi
 
 if [ $PIP_EXIT_CODE -eq 0 ]; then
 success "Python completed"
 else
 if grep -q "angr" "$PIP_LOG" && grep -q "Rust compiler\|can't find Rust" "$PIP_LOG"; then
 warning "angr failed( Rust )"
 echo ""
 info "angr ,Binary analysistool"
 info " angr, Rust:"
 echo " macOS: curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh"
 echo " Ubuntu: curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh"
 echo " : https://rustup.rs/"
 echo ""
 info ",(tool)"
 else
 warning " Python failed,"
 warning ",error"
 echo ""
 info "errordetails( 10 ):"
 tail -n 10 "$PIP_LOG" | sed 's/^/ /'
 echo ""
 fi
 fi
 rm -f "$PIP_LOG"
 else
 warning " requirements.txt, Python "
 fi
}
build_go_project() {
 echo ""
 note "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
 note "⚠️ Go Proxy()"
 note " Proxy : ${GOPROXY}"
 note " config, GOPROXY"
 note "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
 echo ""
 
 info " Go ..."
 GO_DOWNLOAD_LOG=$(mktemp)
 (
 set +e # shelldisablederrorsign out
 export GOPROXY="$GOPROXY"
 go mod download >"$GO_DOWNLOAD_LOG" 2>&1
 echo $? > "${GO_DOWNLOAD_LOG}.exit"
 ) &
 GO_DOWNLOAD_PID=$!
 sleep 0.1
 if kill -0 "$GO_DOWNLOAD_PID" 2>/dev/null; then
 show_progress "$GO_DOWNLOAD_PID" " Go "
 else
 sleep 0.2
 fi
 wait "$GO_DOWNLOAD_PID" 2>/dev/null || true
 
 GO_DOWNLOAD_EXIT_CODE=0
 if [ -f "${GO_DOWNLOAD_LOG}.exit" ]; then
 GO_DOWNLOAD_EXIT_CODE=$(cat "${GO_DOWNLOAD_LOG}.exit" 2>/dev/null || echo "1")
 rm -f "${GO_DOWNLOAD_LOG}.exit" 2>/dev/null || true
 else
 if [ -f "$GO_DOWNLOAD_LOG" ] && grep -q -i "error\|failed" "$GO_DOWNLOAD_LOG" 2>/dev/null; then
 GO_DOWNLOAD_EXIT_CODE=1
 fi
 fi
 rm -f "$GO_DOWNLOAD_LOG" 2>/dev/null || true
 
 if [ $GO_DOWNLOAD_EXIT_CODE -ne 0 ]; then
 error "Go failed"
 exit 1
 fi
 success "Go completed"
 
 info "build..."
 GO_BUILD_LOG=$(mktemp)
 (
 set +e # shelldisablederrorsign out
 export GOPROXY="$GOPROXY"
 go build -o "$BINARY_NAME" cmd/server/main.go >"$GO_BUILD_LOG" 2>&1
 echo $? > "${GO_BUILD_LOG}.exit"
 ) &
 GO_BUILD_PID=$!
 sleep 0.1
 if kill -0 "$GO_BUILD_PID" 2>/dev/null; then
 show_progress "$GO_BUILD_PID" "build"
 else
 sleep 0.2
 fi
 wait "$GO_BUILD_PID" 2>/dev/null || true
 
 GO_BUILD_EXIT_CODE=0
 if [ -f "${GO_BUILD_LOG}.exit" ]; then
 GO_BUILD_EXIT_CODE=$(cat "${GO_BUILD_LOG}.exit" 2>/dev/null || echo "1")
 rm -f "${GO_BUILD_LOG}.exit" 2>/dev/null || true
 else
 if [ -f "$GO_BUILD_LOG" ] && grep -q -i "error\|failed" "$GO_BUILD_LOG" 2>/dev/null; then
 GO_BUILD_EXIT_CODE=1
 fi
 fi
 
 if [ $GO_BUILD_EXIT_CODE -eq 0 ]; then
 success "buildcompleted: $BINARY_NAME"
 rm -f "$GO_BUILD_LOG"
 else
 error "buildfailed"
 echo ""
 info "builderrordetails:"
 cat "$GO_BUILD_LOG" | sed 's/^/ /'
 echo ""
 rm -f "$GO_BUILD_LOG"
 exit 1
 fi
}
need_rebuild() {
 if [ ! -f "$BINARY_NAME" ]; then
 return 0 # build
 fi
 if [ "$BINARY_NAME" -ot cmd/server/main.go ] || \
 [ "$BINARY_NAME" -ot go.mod ] || \
 find internal cmd -name "*.go" -newer "$BINARY_NAME" 2>/dev/null | grep -q .; then
 return 0 # build
 fi
 
 return 1 # build
}
main() {
 info "..."
 check_python
 check_go
 echo ""
 info " Python ..."
 setup_python_env
 echo ""
 if need_rebuild; then
 info "build..."
 build_go_project
 else
 success "Files,build"
 fi
 echo ""
 success "completed!"
 echo ""
 info " Pyntra ..."
 echo "=========================================="
 echo ""
 exec "./$BINARY_NAME"
}
main
