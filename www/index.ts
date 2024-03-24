import {
    createPromiseClient
} from '@connectrpc/connect'
import {
    createConnectTransport,
} from '@connectrpc/connect-web'
import { IpamService } from '../js/api/v1/ipam_connect.ts'
import { CreatePrefixRequest, Prefix } from '../js/api/v1/ipam_pb.ts'


// Make the Ipam Service client
const client = createPromiseClient(
    IpamService,
    createConnectTransport({
        baseUrl: 'http://localhost:9090',
    })
)

// Query for the common elements and cache them.
const containerEl = document.getElementById("conversation-container") as HTMLDivElement;
const inputEl = document.getElementById("user-input") as HTMLInputElement;

// Add an event listener to the input so that the user can hit enter and click the Send button
document.getElementById("user-input")?.addEventListener("keyup", (event) => {
    event.preventDefault();
    if (event.key === "Enter") {
        document.getElementById("send-button")?.click();
        createPrefix()
    }
});

document.getElementById("send-button")?.addEventListener("click", (event) => {
    event.preventDefault();
    createPrefix()
});

// Adds a node to the DOM representing the conversation with Eliza
function addNode(prefix?: Prefix): void {
    const divEl = document.createElement('div');
    const pEl = document.createElement('p');

    const respContainerEl = containerEl.appendChild(divEl);
    respContainerEl.className = `prefix-resp-container`;

    const respTextEl = respContainerEl.appendChild(pEl);
    respTextEl.className = "resp-text";
    if (prefix !== undefined) {
        respTextEl.innerText = prefix.cidr;
    } else {
        respTextEl.innerText = "Unknown CIDR";
    }
}

async function createPrefix() {
    const cidr = inputEl?.value ?? '';

    inputEl.value = '';

    const request = new CreatePrefixRequest({
        cidr
    })
    const response = await client.createPrefix(request)
    addNode(response.prefix);

    console.log("prefix created:" + response.prefix?.cidr)
}
