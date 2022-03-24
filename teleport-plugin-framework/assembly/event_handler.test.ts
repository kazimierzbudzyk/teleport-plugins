import { getFixture, getLatestAPIRequest } from '../vendor/test';
import { events, types } from '../vendor/teleport';
import { handleEvent } from './index';

// Main test function
export function test(): void {
    trace("teleport-plugin-framework tests")

    testRegularEvent();
    testSkipLoginSecretSanta();
    testAddAnnotation();
    testLocking();

    trace("Success!")
}

// Ensure that nomal event passes through
function testRegularEvent(): void {
    const createMeData = getFixture(1)
    assert(handleEvent(createMeData) != null, "Regular event has not been handled")
}

// Ensure that secret santa login hids
function testSkipLoginSecretSanta(): void {
    const loginSecretSantaData = getFixture(2)
    assert(handleEvent(loginSecretSantaData) == null, "Secret-santa login has not been skipped")
}

// Ensure that custom annotation is added to the create access request
function testAddAnnotation(): void {
    const testAccessRequest = getFixture(3)
    const result = handleEvent(testAccessRequest)
    assert(result != null, "Event has not been processed")

    const changedEvent = events.OneOf.decode(result as DataView);
    assert(changedEvent.AccessRequestCreate != null, "AccessRequestCreate is missing")

    const changedAccessRequest = changedEvent.AccessRequestCreate as events.AccessRequestCreate;
    assert(
        changedAccessRequest.Annotations.fields.get("seen-by-us").string_value == "yes", 
        "seen-by-us annotation is not set"
    )
}

function testLocking(): void {
    const loginFoo = getFixture(4)
    handleEvent(loginFoo)
    handleEvent(loginFoo)
    handleEvent(loginFoo)
    const result = handleEvent(loginFoo)
    assert(result != null, "Event has not been processed")

    const request = getLatestAPIRequest()
    assert(request != null, "API request is missing")

    const lock = types.LockV2.decode(request)
    assert(lock.Spec.Target.User == "foo", "Lock user foo has not been generated")
}