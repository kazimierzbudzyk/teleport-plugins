// protobuf classes for host message exchange
import { events, google, types } from '../vendor/teleport';
// memory data store for event count calculations
import * as store from '../vendor/store';
// API methods
import { upsertLock } from '../vendor/api';

const maxFailedLoginAttempts = 3;     // 3 tries
const failedAttemptsTimeout = 60 * 5; // within 5 minutes

type Event = events.OneOf;

// handleEvent is the main plugin function
export function handleEvent(source: Event): Event | null {
    var event:Event | null = source;

    event = hideEvent(event);
    if (event == null) {
        return null;
    }

    event = addRequiredLabels(event);
    if (event == null) {
        return null;
    }

    event = createLockBasedOnEvent(event);
    if (event == null) {
        return null;
    }

    return event;
}

// Hides secret-santa user logins
function hideEvent(event: Event): Event | null {
    if (event.UserLogin != null) {
        const userLogin = event.UserLogin as events.UserLogin
        if (userLogin.User.Login == "secret-santa") {
            return null;
        }
    }

    return event;
}

// Adds label to access request create
function addRequiredLabels(event: Event): Event | null {
    if (event.AccessRequestCreate != null) {
        const request = event.AccessRequestCreate as events.AccessRequestCreate;

        const value = new google.protobuf.Value()
        value.string_value = "yes"

        request.Annotations.fields.set("seen-by-us", value)
    }

    if (event.RoleCreate != null) {
        const roleCreate = event.RoleCreate as events.RoleCreate;

        roleCreate.Metadata.ClusterName = "changed-cluster-name"
    }

    return event;
}

function createLockBasedOnEvent(event: Event): Event | null {
    if (event.UserLogin != null) {
        const login = event.UserLogin as events.UserLogin;
        const count = store.takeToken(login.User.Login, failedAttemptsTimeout) // 5 minutes
        if (count > maxFailedLoginAttempts) {
            trace("Suspicious login activity detected, attempts made:", 1, count)

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

            const encoded = lock.encode()
            upsertLock(changetype<usize>(encoded))
        }
    }

    return event;
}