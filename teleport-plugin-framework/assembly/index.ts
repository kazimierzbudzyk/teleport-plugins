import { events } from '../vendor/teleport';
import { handleEvent as handleEventActual } from './event_handler'

export { __protobuf_alloc, __protobuf_getAddr, __protobuf_getLength } from "../vendor/teleport"

// handleEvent is the event handler plugin entrypoint
export function handleEvent(eventData: DataView): DataView | null {
    let event:events.OneOf | null = events.OneOf.decode(eventData);
    if (event == null) {
        throw new Error("Failed to decode Event from protobuf!")
    }

    event = handleEventActual(event)
    if (event == null) {
        return null
    }

    return event.encode()
}            
