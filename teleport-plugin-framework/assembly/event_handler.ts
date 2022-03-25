// protobuf classes for host message exchange
import { events, google } from '../vendor/teleport';
// memory data store for event count calculations
import * as store from '../vendor/store';
// API methods
import { createLock } from '../vendor/api';

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

        // If a login was not successful        
        if (login.Status.Success == false) {
            // Record login attempt and get current attempts within the time frame for a user
            const count = store.takeToken(login.User.Login, failedAttemptsTimeout) // 5 minutes

            // If limit is exceeded
            if (count > maxFailedLoginAttempts) {
                trace("Suspicious login activity detected, attempts made:", 1, count)

                // Create lock
                createLock(login.User, 3600)
            }
        }
    }

    return event;
}