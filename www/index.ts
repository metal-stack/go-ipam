import {
    createPromiseClient
} from '@connectrpc/connect'
import {
    createConnectTransport,
} from '@connectrpc/connect-web'
import { IpamService } from '../js/api/v1/ipam_connect.ts'
import { CreatePrefixRequest, ListPrefixesRequest, Prefix } from '../js/api/v1/ipam_pb.ts'


// Make the Ipam Service client
const client = createPromiseClient(
    IpamService,
    createConnectTransport({
        baseUrl: 'http://localhost:9090',
    })
)

// Query for the common elements and cache them.
const containerEl = document.getElementById("prefix-container") as HTMLDivElement;
const cidrInputEl = document.getElementById("cidr-input") as HTMLInputElement;

// Add an event listener to the input so that the user can hit enter and click the Send button
document.getElementById("cidr-input")?.addEventListener("keyup", (event) => {
    event.preventDefault();
    if (event.key === "Enter") {
        createPrefix()
    }
});

document.getElementById("create-prefix-button")?.addEventListener("click", (event) => {
    event.preventDefault();
    createPrefix()
});

function refreshPrefixes(prefixes: Prefix[]): void {
    const divEl = document.createElement('div');
    const pEl = document.createElement('p');

    const respContainerEl = containerEl.appendChild(divEl);
    respContainerEl.className = `prefix-resp-container`;

    const respTextEl = respContainerEl.appendChild(pEl);
    respTextEl.className = "resp-text";

    for (let prefix of prefixes) {
        if (prefix !== undefined) {
            respTextEl.innerText = prefix.cidr;
        } else {
            respTextEl.innerText = "Unknown CIDR";
        }
    }
}

async function listPrefixes() {
    const request = new ListPrefixesRequest({})
    const response = await client.listPrefixes(request)
    refreshPrefixes(response.prefixes)
}

async function createPrefix() {
    const cidr = cidrInputEl?.value ?? '';

    cidrInputEl.value = '';

    const request = new CreatePrefixRequest({
        cidr
    })
    const response = await client.createPrefix(request)
    listPrefixes();
    console.log("prefix created:" + response.prefix?.cidr)
}