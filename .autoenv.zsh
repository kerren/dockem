echo "Entering dockem repository"

set -a
source .env || true
set +a

echo " [+] Sourced the .env file for testing purposes"
echo " [+] Using the docker username: \"${DOCKER_USERNAME}\""
