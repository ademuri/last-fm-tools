set -euo pipefail

tools_dir="$HOME/.last-fm-tools"
mkdir -p "${tools_dir}"
cd "${tools_dir}"

if [ ! -d "last-fm-tools" ]; then
  git clone https://github.com/ademuri/last-fm-tools.git
fi

cd last-fm-tools
git pull

bazel run //:last-fm-tools -- send-reports
