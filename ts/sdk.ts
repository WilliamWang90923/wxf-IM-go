

export const Ping = new Uint8Array([0, 100, 0, 0, 0, 0])
export const Pong = new Uint8Array([0, 101, 0, 0, 0, 0])

const HEARTBEAT_INTERVAL = 10

export const sleep = async (second: number): Promise<void> => {
    return new Promise((resolve) => {
        setTimeout(() => {
            resolve()
        }, second * 1000)
    })
}