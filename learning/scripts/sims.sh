#!/usr/bin/env bash
set -e

# --- Configuration ---
# RabbitMQ management credentials and endpoint
USER="${RABBITMQ_USER:-komod_admin}"
PASS="${RABBITMQ_PASS:-universal}"
HOST="${RABBITMQ_HOST:-localhost:15672}"
VHOST="${RABBITMQ_VHOST:-komod}"
API="http://${HOST}/api"
CRED="${USER}:${PASS}"
CTYPE="content-type:application/json"

EXCHANGES=("user.events" "auth.events" "wallet.events")
QUEUES=("q.wallet.create_deposit" "q.message.all_notifications")

check_connectivity() {
  local conn_status
  conn_status=$(curl -s -o /dev/null -w "%{http_code}" -u "$CRED" "${API}/whoami" 2>/dev/null || echo "000")
  if [[ "$conn_status" == "000" ]]; then
    echo "❌ Error: Cannot connect to RabbitMQ Management API at ${HOST}." >&2
    echo "   Please make sure RabbitMQ is running." >&2
    exit 1
  elif [[ "$conn_status" == "401" ]]; then
    echo "❌ Error: RabbitMQ authentication failed for user '${USER}'." >&2
    exit 1
  fi
}

check_infrastructure() {
  check_connectivity

  local missing=()

  # Check Vhost (query /api/vhosts which is accessible to both management and admin roles)
  local vhosts_json
  vhosts_json=$(curl -s -u "$CRED" "${API}/vhosts" 2>/dev/null || echo "[]")
  if ! echo "$vhosts_json" | grep -F -q "\"name\":\"${VHOST}\""; then
    missing+=("VHost '${VHOST}'")
  else
    # Check Topic Exchanges
    for ex in "${EXCHANGES[@]}"; do
      local ex_status
      ex_status=$(curl -s -o /dev/null -w "%{http_code}" -u "$CRED" "${API}/exchanges/${VHOST}/${ex}" 2>/dev/null || echo "000")
      if [[ "$ex_status" != "200" ]]; then
        missing+=("Exchange '${ex}'")
      fi
    done

    # Check Queues
    for q in "${QUEUES[@]}"; do
      local q_status
      q_status=$(curl -s -o /dev/null -w "%{http_code}" -u "$CRED" "${API}/queues/${VHOST}/${q}" 2>/dev/null || echo "000")
      if [[ "$q_status" != "200" ]]; then
        missing+=("Queue '${q}'")
      fi
    done

    # Check Bindings
    local bindings=(
      "e/user.events/q/q.wallet.create_deposit:user.registered"
      "e/user.events/q/q.message.all_notifications:#"
      "e/auth.events/q/q.message.all_notifications:#"
      "e/wallet.events/q/q.message.all_notifications:#"
    )

    for b in "${bindings[@]}"; do
      local endpoint="${b%%:*}"
      local routing_key="${b##*:}"
      local binding_resp
      binding_resp=$(curl -s -u "$CRED" "${API}/bindings/${VHOST}/${endpoint}" 2>/dev/null || echo "[]")
      if ! echo "$binding_resp" | grep -F -q "\"routing_key\":\"${routing_key}\""; then
        missing+=("Binding '${endpoint}' with routing key '${routing_key}'")
      fi
    done
  fi

  if [[ ${#missing[@]} -gt 0 ]]; then
    echo "⚠️  WARNING: RabbitMQ infrastructure requirements are not met!" >&2
    echo "The following required components are missing:" >&2
    for item in "${missing[@]}"; do
      echo "  - ${item}" >&2
    done
    echo "" >&2
    echo "Please run this command with '--create-requirements' first:" >&2
    echo "  $0 --create-requirements" >&2
    echo "" >&2
    return 1
  fi

  return 0
}

create_infrastructure() {
  check_connectivity

  echo "🚀 Building RabbitMQ Architecture..."
  echo "------------------------------------------------"

  # 1. Create Vhost if not present
  local vhosts_json
  vhosts_json=$(curl -s -u "$CRED" "${API}/vhosts" 2>/dev/null || echo "[]")
  if ! echo "$vhosts_json" | grep -F -q "\"name\":\"${VHOST}\""; then
    echo "[1/4] Creating VHost '${VHOST}'..."
    curl -s -f -u "$CRED" -X PUT "${API}/vhosts/${VHOST}" > /dev/null
  else
    echo "[1/4] VHost '${VHOST}' already exists."
  fi

  # 2. Create Topic Exchanges
  echo "[2/4] Creating Topic Exchanges..."
  for EX in "${EXCHANGES[@]}"; do
    curl -s -f -u "$CRED" -H "$CTYPE" -X PUT -d '{"type":"topic","durable":true}' "${API}/exchanges/${VHOST}/${EX}" > /dev/null
  done

  # 3. Create Queues
  echo "[3/4] Creating Queues..."
  for Q in "${QUEUES[@]}"; do
    curl -s -f -u "$CRED" -H "$CTYPE" -X PUT -d '{"durable":true}' "${API}/queues/${VHOST}/${Q}" > /dev/null
  done

  # 4. Bind Queues to Exchanges using Routing Keys
  echo "[4/4] Creating Queue Bindings..."
  # Wallet Service only cares about new users
  curl -s -f -u "$CRED" -H "$CTYPE" -X POST -d '{"routing_key":"user.registered"}' "${API}/bindings/${VHOST}/e/user.events/q/q.wallet.create_deposit" > /dev/null

  # Message Service cares about EVERYTHING (using the '#' wildcard)
  curl -s -f -u "$CRED" -H "$CTYPE" -X POST -d '{"routing_key":"#"}' "${API}/bindings/${VHOST}/e/user.events/q/q.message.all_notifications" > /dev/null
  curl -s -f -u "$CRED" -H "$CTYPE" -X POST -d '{"routing_key":"#"}' "${API}/bindings/${VHOST}/e/auth.events/q/q.message.all_notifications" > /dev/null
  curl -s -f -u "$CRED" -H "$CTYPE" -X POST -d '{"routing_key":"#"}' "${API}/bindings/${VHOST}/e/wallet.events/q/q.message.all_notifications" > /dev/null

  echo "✅ Infrastructure built successfully."
  echo ""
  echo "You can now run the simulation:"
  echo "  $0"
}

run_simulation() {
  echo "🚀 Starting RabbitMQ Microservices Simulation..."
  echo "------------------------------------------------"

  # --- 1. Simulating User Registration ---
  echo "[1/3] User Service: Publishing 'user.registered' event..."
  if curl -s -u "$CRED" -H "$CTYPE" -X POST \
    -d '{"properties":{},"routing_key":"user.registered","payload":"{\"userId\": 1045, \"name\": \"Alice\", \"email\": \"alice@example.com\"}","payload_encoding":"string"}' \
    "${API}/exchanges/${VHOST}/user.events/publish" | grep -q '"routed":true'; then
    echo "✅ Event shouted to user.events exchange."
  else
    echo "❌ Failed to route event to user.events exchange." >&2
    exit 1
  fi
  echo ""
  sleep 1

  # --- 2. Wallet Service Processing ---
  echo "[2/3] Wallet Service: Consuming event and creating deposit..."
  # Read the event from the wallet queue and acknowledge it
  curl -s -u "$CRED" -H "$CTYPE" -X POST \
    -d '{"count":1,"ackmode":"ack_requeue_false","encoding":"auto"}' \
    "${API}/queues/${VHOST}/q.wallet.create_deposit/get" > /dev/null

  # Publish a new event stating the deposit was created
  if curl -s -u "$CRED" -H "$CTYPE" -X POST \
    -d '{"properties":{},"routing_key":"wallet.deposit.created","payload":"{\"userId\": 1045, \"walletId\": \"W-8892\", \"initialBalance\": 0}","payload_encoding":"string"}' \
    "${API}/exchanges/${VHOST}/wallet.events/publish" | grep -q '"routed":true'; then
    echo "✅ Event shouted to wallet.events exchange."
  else
    echo "❌ Failed to route event to wallet.events exchange." >&2
    exit 1
  fi
  echo ""
  sleep 1

  # --- 3. Message Service Processing ---
  echo "[3/3] Message Service: Checking omniscient queue for notifications..."
  # The Message Service should have TWO messages waiting:
  # 1. The original user.registered event
  # 2. The subsequent wallet.deposit.created event
  echo "Waiting messages in q.message.all_notifications:"
  if command -v jq >/dev/null 2>&1; then
    curl -s -u "$CRED" -H "$CTYPE" -X POST \
      -d '{"count":2,"ackmode":"ack_requeue_false","encoding":"auto"}' \
      "${API}/queues/${VHOST}/q.message.all_notifications/get" | jq -r '.[].payload // empty'
  else
    curl -s -u "$CRED" -H "$CTYPE" -X POST \
      -d '{"count":2,"ackmode":"ack_requeue_false","encoding":"auto"}' \
      "${API}/queues/${VHOST}/q.message.all_notifications/get" | grep -o '"payload":"[^"]*"' | sed 's/"payload"://g'
  fi

  echo ""
  echo "🎉 Simulation Complete!"
}

user_new(){
    echo "Adding new user"
    
    if curl -s -u "$CRED" -H "$CTYPE" -X POST \
        -d '{"properties":{},"routing_key":"user.registered","payload":"{\"userId\": '$RANDOM', \"name\": \"'$1'\", \"email\": \"'$2'\"}","payload_encoding":"string"}' \
    "${API}/exchanges/${VHOST}/user.events/publish" | grep -q '"routed":true'; then
    echo "✅ Event shouted to user.events exchange."
  else
    echo "❌ Failed to route event to user.events exchange." >&2
    exit 1
  fi
}

show_help() {
  echo "Usage: $0 [OPTION]"
  echo ""
  echo "Options:"
  echo "  (no args)               Check if infrastructure exists, then run message simulation"
  echo "  --add-user <name> <email> Create new user"
  echo "  --create-requirements   Build RabbitMQ vhost, exchanges, queues, and bindings"
  echo "  -h, --help              Show this help message"
}

# --- Main Entrypoint ---
case "$1" in
  --add-user|-u)
    user_new "$2" "$3"
    ;;
  --create-requirements|-c)
    create_infrastructure
    ;;
  --help|-h)
    show_help
    ;;
  "")
    if check_infrastructure; then
      run_simulation
    else
      exit 1
    fi
    ;;
  *)
    echo "❌ Unknown option: $1" >&2
    show_help >&2
    exit 1
    ;;
esac
