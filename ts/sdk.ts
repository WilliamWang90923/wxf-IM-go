const WebSocket = require('ws')

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

export enum ACK {
    Success = "SUCCESS",
    Timeout = "TIMEOUT",
    LoginFailed = "LOGINFAIL",
    Logined = "LOGINED"
}

export enum State {
    INIT,
    CONNECTING,
    CONNECTED,
    RECONNECTING,
    CLOSEING,
    CLOSED,
}

export let doLogin = async (url: string): Promise<{ status: string, conn: WebSocket}> => {
    const LoginTimeout = 5
    return new Promise((resolve, reject) => {
        let conn = new WebSocket(url)
        conn.binaryType = "arraybuffer"

        let loginTimeout = setTimeout(() => {
            resolve({
                status: ACK.Timeout,
                conn: conn
            });
        }, LoginTimeout * 1000);

        conn.onopen = () => {
            console.info("websocket open - ready state: ", conn.readyState)

            if (conn.readyState === WebSocket.OPEN) {
                clearTimeout(loginTimeout)
                resolve({
                    status: ACK.Success,
                    conn: conn
                });
            }
        }
        conn.onerror = () => {
            clearTimeout(loginTimeout)
            resolve({
                status: ACK.LoginFailed,
                conn: conn
            });
        }
    })
}

export class IMClient {
    ws_url: string
    state = State.INIT
    private conn: WebSocket | null
    private lastRead: number

    constructor(url: string, user: string) {
        this.ws_url = `${url}?user=${user}`
        this.conn = null
        this.lastRead = Date.now()
    }

    async login(): Promise<{ status: string }> {
        if (this.state == State.CONNECTED) {
            return {status: ACK.Logined}
        }
        this.state = State.CONNECTING

        let { status, conn } = await doLogin(this.ws_url)
        console.info("login - ", status)

        conn.onmessage = (evt: MessageEvent) => {
            this.lastRead = Date.now()
            try {
                let buf = Buffer.from(evt.data)
                let command = buf.readInt16BE(0)
                let len = buf.readInt32BE(2)
                console.info(`<<<< received a message ; command:${command} len: ${len}`)
                if (command == 101) {
                    console.info("<<<< received a pong...")
                }
            } catch (e) {
                console.error(evt.data, e)
            }
        }

        conn.onerror = (err:Event) => {
            console.error("websocket error: ", err)
            // todo handle error
            this.errorHandler(new Error(err.type))
        }
        conn.onclose = (event: CloseEvent) => {
            if (this.state == State.CLOSEING) {
                // todo
                this.onclose("log out")
                return
            }
            this.errorHandler(new Error(event.reason))
        }
        this.conn = conn
        this.state = State.CONNECTED

        this.heartbeatloop()
        this.readDeadlineLoop()

        return { status }
    }

    logout() {
        if (this.state === State.CLOSEING) {
            return
        }
        this.state = State.CLOSEING
        if (!this.conn) {
            return
        }
        console.info("Connection closing...")
        this.conn.close()
    }

    private heartbeatloop() {
        console.debug("heart beat loop start...")

        let loop = () => {
            if (this.state != State.CONNECTED) {
                console.debug("heartbeatLoop exited")
                return
            }
            console.log(`>>> send ping ; state is ${this.state},`)
            this.send(Ping)

            setTimeout(loop, HEARTBEAT_INTERVAL * 1000)
        }
        setTimeout(loop, HEARTBEAT_INTERVAL * 1000)
    }

    private readDeadlineLoop() {
        console.debug("deadline loop start...")

        let loop = () => {
            if (this.state != State.CONNECTED) {
                console.debug("deadlineLoop exited")
                return
            }
            if ((Date.now() - this.lastRead) > 3 * HEARTBEAT_INTERVAL * 1000) {
                // timeout errorHandler
                this.errorHandler(new Error("read timeout"))
            }
            setTimeout(loop, 1000)
        }
        setTimeout(loop, 1000)
    }

    private onclose(reason: string) {
        console.info("connection closed due to " + reason)
        this.state = State.CLOSED
    }

    private send(data: Uint8Array | Buffer) {
        if (this.conn == null) {
            return false
        }
        try {
            this.conn.send(data)
        } catch (err) {
            this.errorHandler(new Error("read timeout"))
            return false
        }
        return true
    }

    private async errorHandler(error: Error) {
        if (this.state == State.CLOSED || this.state == State.CLOSEING) {
            return
        }
        this.state = State.RECONNECTING
        console.info("encounted error: " + error + ", try reconnecting...")
        for (let i = 0; i < 5; i++) {
            let { status } = await this.login()
            if (status == "Success") {
                return
            }
            await sleep(4)
        }
        this.onclose("reconnect timeout")
    }
}