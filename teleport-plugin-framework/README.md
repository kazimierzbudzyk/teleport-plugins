# Teleport WASM plugin framework

Teleport WASM plugin framework allows to manipulate events forwarded by `event-handler`.

# Prerequisites

* go 1.17+
* node >= 16.3 < 17 (please, use [nvm](https://github.com/nvm-sh/nvm) to install specific version)
* yarn (check [YARN installation instructions](https://classic.yarnpkg.com/lang/en/docs/install/))

# Installation

## Conventional architecture

Run:

```sh
git clone git@github.com/gravitational/teleport-plugins.git --branch feature/wasm
cd teleport-plugin-framework
make build
```

## Mac M1

Install [Rust](https://www.rust-lang.org/tools/install).

Run:

```sh
git clone git@github.com/gravitational/teleport-plugins.git --branch feature/wasm
cd teleport-plugin-framework
make get-custom-wasmer-runtime
CUSTOM_WASMER_RUNTIME=true make build
```

# Start the new plugin from scratch

## Generate the plugin

Run the following command to generate the minimal plugin setup:

```sh
./build/teleport-plugin-framework new ~/teleport-plugin
```

The tool will check if required node and yarn version is installed in the system, will create file structure for the example plugin and will run `yarn install` in the target folder.

## Check the generated plugin

Run:

```sh
cd ~/teleport-plugin && yarn test
```

This will show the following output:

```sh
INFO   teleport-plugin-framework tests  wasm/assembly_script_env.go:146
INFO   Suspicious login activity detected, attempts made: 4 wasm/assembly_script_env.go:146
INFO   Success!  wasm/assembly_script_env.go:146
```

Meaning that the setup was successful.

# How plugins work

WASM plugins can be written in any language which supports [WASM as a target](https://webassembly.org/getting-started/developers-guide/). The most reliable choices are [Rust](https://rust-lang.org), C/C++ and [AssemblyScript](https://assemblyscript.org/). We chose [AssemblyScript](https://assemblyscript.org/) as it has the simplest TypeScript-linke syntax and tiny runtime compared to the most available WASM languages.

## The folder structure

Every plugin consists of the following folders:

* `assembly` - the plugin code and tests.
* `build` - the target WASM binaries.
* `fixtures` - the test fixtures.
* `vendor` - external method definitions and helper code.

## The host-plugin message exchange

Teleport API uses [protocol buffers](https://developers.google.com/protocol-buffers). It is the main protocol for message exchange in Teleport. We use the same protocol to talk to plugin. For now, plugins are supported by `event-handler` only. This tool is used to forward Teleport cluster events to fluentd. WASM plugins can manipulate events in the middle.

The pipeline is the following:
1. Event gets received by `event-handler` (it either can be main log event or session event).
2. `event-handler` encodes an event to the protobuf binary message, copies it to the WASM memory and calls the `handleEvent` method.
3. `handleEvent` method decodes binary message into the counterpart `AssemblyScript` class instance.
4. After the processing is done, `handleEvent` encodes the result into protobuf binary message and returns it. 
5. `event-handler` decodes the result and sends it to fluentd. If a result is nil, message is skipped.

# Minimal event plugin and test

The following code receives an upcoming event and returns it unchanged. Put it to `assembly/index.ts`.

```ts
// event protobuf class
import { events } from '../vendor/teleport';

// These methods are used by the host for sending message to the plugin side
export {
    __protobuf_alloc,
    __protobuf_getAddr,
    __protobuf_getLength,
} from '../vendor/teleport';

// Type alias for an event
type Event = events.OneOf;

// plugin entry point
export function handleEvent(eventData: DataView): DataView | null {
    // decode the event from it's binary form
    let event:Event | null = events.OneOf.decode(eventData);
    if (event == null) {
        throw new Error("Failed to decode Event from protobuf!")
    }

    // encode an event back and return it
    return event.encodeDataView();
}
```

Now, put the following content into `assembly/index.test.ts`:

```ts
import { getFixture } from '../vendor/test';
import { events } from '../vendor/teleport';
import { handleEvent } from '.';

export { 
    __protobuf_alloc,
    __protobuf_getAddr,
    __protobuf_getLength,
} from '../vendor/teleport';

export { handleEvent } from './index';

// Main test function
export function test(): void {
    trace("teleport-plugin-framework tests")

    // Get test event from the fixture #1 (see below)
    const createMeData = getFixture(1)

    // Send test event to the plugin
    const result = handleEvent(createMeData)
    assert(result != null, "Event is not handled")

    // Ensure that user login has not been changed
    const resultingEvent = events.OneOf.decode(result as DataView);
    assert(resultingEvent.UserCreate != null, "Event has changed")

    // Ensuret that login has not been changed
    const userCreateEvent = resultingEvent.UserCreate as events.UserCreate
    assert(userCreateEvent.User.Login == "foo", "Login has changed")

    trace("Success!")
}
```

Run:

```sh
yarn test
```

You should see the following output:

```sh
INFO   teleport-plugin-framework tests  wasm/assembly_script_env.go:146
INFO   Success!  wasm/assembly_script_env.go:146
✨  Done in 2.23s.
```

Tests are run in the same environment and follow the same message exchange pipeline as the production environment do.

# Test fixtures

Test events are loaded from json files located in the `fixtures` folder. The fixture file name has the following template: `<id>-<description>.json` where id is the sequential fixture number.

## Fixture file structure

The fixture has the following structure:

```json
{
  "name": "events.user_create",
  "type": "types.OneOf",
  "description": "User login event",
  "data": {
    "UserCreate": {
      "Metadata": {
        "Index": 1,
        "ID": "36da4091-eb14-f814-69f8-5b97bd919e3d",
        "Type": "user.create",
        "ClusterName": "test-cluster",
        "Time": "2021-12-17T17:11:33+03:00"
      },
      "User": {
        "Login": "foo",
        "User": "foo"
      },
      "Resource": {
        "Name": "foo"
      },
      "Roles": [
        "test-user"
      ]
    }
  }
}
```

* `name` - internal fixture type name
* `type` - the protobuf message type name (would be `types.OneOf` for all events).
* `description` - fixture description
* `data` - fixture data

## Generating the fixture file

In the plugin folder, run the following command:

```sh
./teleport-plugin-framework fixtures list-templates
```

You will see the list of available fixture templates:

```sh
./teleport-plugin-framework fixtures generate events.user_create <name> - Create user event (types.OneOf)
./teleport-plugin-framework fixtures generate events.user_login <name> - User login event (types.OneOf)
```

To generate the fixture file from a template, choose the required type from the list and run:

```sh
./teleport-plugin-framework fixtures generate events.user_create supervisor
```

Where `supervisor` is the description.

You will get the following response:

```sh
Fixture generated: fixtures/5-supervisor.json
```

Now, you can open the file and change fixture data manually.

## Using the fixture file

To load a fixture in a test, use the following method:

```ts
const createSupervisorUser = getFixture(5) // Where 5 is the fixture number
```

Now, you can send this data to the `handlePlugin` method:

```ts
const result = handleEvent(createSupervisorUser)
```

# Skipping events

Let's say we want to skip logins from a user named `secret-santa`.

Put the following code into `assembly/index.ts`:

```ts
// event protobuf class
import { events } from '../vendor/teleport';

// This methods are used by the host for message exchange
export {
    __protobuf_alloc,
    __protobuf_getAddr,
    __protobuf_getLength,
} from '../vendor/teleport';

// Type alias for an event
type Event = events.OneOf;

// plugin entry point
export function handleEvent(eventData: DataView): DataView | null {
    let event:Event | null = events.OneOf.decode(eventData);
    if (event == null) {
        throw new Error("Failed to decode Event from protobuf!")
    }

    // If this is the login event
    if (event.UserLogin != null) {
        const userLogin = event.UserLogin as events.UserLogin
        if (userLogin.User.Login == "secret-santa") { // And the login is `secret-santa`
            return null; // Skip the event
        }
    }

    return event.encodeDataView();
}
```

Fixture #2 in the test setup represents the login event of `secret-santa` user. Given that, put the following in `assembly/index.test.ts`:

```ts
import { getFixture } from '../vendor/test';
import { handleEvent } from '.';

export { 
    __protobuf_alloc,
    __protobuf_getAddr,
    __protobuf_getLength,
} from '../vendor/teleport';

export { handleEvent } from './index';

// Main test function
export function test(): void {
    trace("teleport-plugin-framework tests")

    const loginSecretSantaData = getFixture(2)
    assert(handleEvent(loginSecretSantaData) == null, "secret-santa login was not skipped")

    trace("Success!")
}
```

And run: `yarn test`.

# Modifying events

Let's say we want to add a label `'seen-by-us': 'yes'` to all access requests.

Put the following contents in the `assembly/index.ts`:

```ts
// event protobuf class
import { google, events } from '../vendor/teleport';

// This methods are used by the host for message exchange
export {
    __protobuf_alloc,
    __protobuf_getAddr,
    __protobuf_getLength,
} from '../vendor/teleport';

// Type alias for an event
type Event = events.OneOf;

// plugin entry point
export function handleEvent(eventData: DataView): DataView | null {
    let event:Event | null = events.OneOf.decode(eventData);
    if (event == null) {
        throw new Error("Failed to decode Event from protobuf!")
    }

    // If an event is AccessRequestCreate
    if (event.AccessRequestCreate != null) {
        const request = event.AccessRequestCreate as events.AccessRequestCreate;

        // Add label
        const value = new google.protobuf.Value()
        value.string_value = "yes"

        request.Annotations.fields.set("seen-by-us", value)
    }

    return event.encodeDataView();
}
```

Fixture #3 in the test setup represents AccessRequest create event. Given that, put the following contents into `assembly/index.test.ts`:

```ts
import { events } from '../vendor/teleport';
import { getFixture } from '../vendor/test';
import { handleEvent } from '.';

export { 
    __protobuf_alloc,
    __protobuf_getAddr,
    __protobuf_getLength,
} from '../vendor/teleport';

export { handleEvent } from './index';

// Main test function
export function test(): void {
    trace("teleport-plugin-framework tests")

    const testAccessRequest = getFixture(3)
    const result = handleEvent(testAccessRequest)
    assert(result != null, "Event has not been processed")

    const changedEvent = events.OneOf.decode(result as DataView);
    assert(changedEvent.AccessRequestCreate != null, "AccessRequestCreate is present")

    const changedAccessRequest = changedEvent.AccessRequestCreate as events.AccessRequestCreate;
    assert(
        changedAccessRequest.Annotations.fields.get("seen-by-us").string_value == "yes", 
        "seen-by-us annotation is not set"
    )

    trace("Success!")
}
```

Run: `yarn test`.

# Counting events and calling Teleport API

Let's say we want to lock user if he fails to login 3 times within latest 5 minutes.

Put the following content into `assembly/index.ts`:

```ts
import { google, events, types } from '../vendor/teleport';
import * as store from '../vendor/store';
import { upsertLock } from '../vendor/api';

// This methods are used by the host for message exchange
export {
    __protobuf_alloc,
    __protobuf_getAddr,
    __protobuf_getLength,
} from '../vendor/teleport';

// Type alias for an event
type Event = events.OneOf;

const maxFailedLoginAttempts = 3;     // 3 tries
const failedAttemptsTimeout = 60 * 5; // within 5 minutes

// plugin entry point
export function handleEvent(eventData: DataView): DataView | null {
    let event:Event | null = events.OneOf.decode(eventData);
    if (event == null) {
        throw new Error("Failed to decode Event from protobuf!")
    }

    if (event.UserLogin != null) {
        const login = event.UserLogin as events.UserLogin;

        // If a login was not successful
        if (login.Status.Success == false) {
            // Record login attempt and get current attempts within the time frame for a user
            const count = store.takeToken(login.User.Login, failedAttemptsTimeout) // 5 minutes

            // If limit is exceeded
            if (count > maxFailedLoginAttempts) {
                trace("Suspicious login activity detected, attempts made:", 1, count)

                // Create lock object
                const lock = new types.LockV2()
                lock.Metadata = new types.Metadata()
                lock.Metadata.Name = "wasm-plugin-lock-" + login.User.Login;

                lock.Spec = new types.LockSpecV2()
                lock.Spec.Message = "Suspicious login"
                lock.Spec.Target = new types.LockTarget()
                lock.Spec.Target.Login = login.User.Login;
                lock.Spec.Target.User = login.User.User;
                
                lock.Spec.Expires = new google.protobuf.Timestamp()
                lock.Spec.Expires.seconds = Date.now() + 3600;

                // Call Teleport API
                const encoded = lock.encodeDataView()
                upsertLock(changetype<usize>(encoded))
            }
        }
    }

    return event.encodeDataView();
}
```

Fixture #4 in the test setup represents failed login attempt. Given that, put the following contents into `assembly/index.test.ts`:

```ts
import { types } from '../vendor/teleport';
import { getFixture, getLatestAPIRequest } from '../vendor/test';
import { handleEvent } from '.';

export { 
    __protobuf_alloc,
    __protobuf_getAddr,
    __protobuf_getLength,
} from '../vendor/teleport';

export { handleEvent } from './index';

// Main test function
export function test(): void {
    trace("teleport-plugin-framework tests")

    const loginFoo = getFixture(4)
    handleEvent(loginFoo) // Failed attempt #1
    handleEvent(loginFoo) // Failed attempt #2
    handleEvent(loginFoo) // Failed attempt #3
    
    const result = handleEvent(loginFoo) // Failed attempt #4, user should be locked at this point
    assert(result != null, "Event has not been processed")

    const request = getLatestAPIRequest()
    assert(request != null, "API request is missing")

    const lock = types.LockV2.decode(request)
    assert(lock.Spec.Target.User == "foo", "Lock user foo has not been generated")

    trace("Success!")
}
```

And run `yarn test`.

Please note `getLatestAPIRequest()` method call. Tests call mock API. Mock API methods are always successful. Latest API request is saved in memory and returned by this method. In our example, this method will return binary representation of `types.LockV2` object constructed in `index.ts`.

# Connecting the plugin to `event-handler`

## Prepare the plugin



## Installation

Build `event-handler` plugin from source.

### Conventional architecture

```sh
git clone git@github.com/gravitational/teleport-plugins.git --branch feature/wasm
cd event-handler
make build
```

### Mac M1

Install [Rust](https://www.rust-lang.org/tools/install).

```sh
git clone git@github.com/gravitational/teleport-plugins.git --branch feature/wasm
cd event-handler
make get-custom-wasmer-runtime
CUSTOM_WASMER_RUNTIME=true make build
```

## Configuration

Please, follow the [official documentation](https://goteleport.com/docs/setup/guides/fluentd/#step-26-generate-configuration).

After you finished setting everything up, please run:

```sh
cd ~/teleport-plugin
yarn asbuild
```

After that, add the following lines to `teleport-event-handler.toml`:

```toml
[wasm]
plugin = "~/teleport-plugin/build/production.wasm"
```

Start `event-handler`. Events would be processed via the plugin.