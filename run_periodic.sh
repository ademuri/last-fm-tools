set -euo pipefail

# Send reports for these users to these email addresses (in order).
users=("me")
emails=("me@me.com")

mkdir -p "$HOME/.last-fm-tools"
cd "$HOME/.last-fm-tools"

if [ ! -d "last-fm-tools" ]; then
  git clone https://github.com/ademuri/last-fm-tools.git
fi

cd last-fm-tools
git pull

for i in ${!users[@]}; do
  echo "Updating database for ${users[$i]}"
  bazel run //:last-fm-tools -- update --user "${users[$i]}" --database "${PWD}/lastfm.db"
  bazel run //:last-fm-tools -- email "${emails[$i]}" top-artists new-artists --user "${users[$i]}" --database "${PWD}/lastfm.db" --name "Artists" --run_day 1
  bazel run //:last-fm-tools -- email "${emails[$i]}" top-albums new-albums --user "${users[$i]}" --database "${PWD}/lastfm.db" --name "Albums" --run_day 15
done

