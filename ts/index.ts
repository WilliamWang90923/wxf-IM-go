import {IMClient} from "./sdk";


const main = async () => {
    let cli = new IMClient("ws://localhost:8001", "wxf");
    let { status } = await cli.login();
    console.log("client login..., status: ", status)
}

main()