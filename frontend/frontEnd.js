let numberMessage = 0
let myID = ""
let active = "Rumors"

function sendNewMessage() {
    let inputText = document.getElementById("text-input")


    if (inputText.value != "") {
        $.ajax({
            type: "POST",
            url: "http://localhost:8080/message",
            data: {
                "value": inputText.value,
                "dest": active
            },
            success: () => {
                inputText.value = ""
            }, error: (status) => {
                console.log(status)
                inputText.value = ""
            }
        })
    } else {
        alert("Message cannot be empty")
    }


}


function addNewMessage(origin, content) {
    let messageList = document.getElementById("chat-message-list")
    let messageRow = document.createElement("div")
    if (origin === myID) {
        messageRow.className = "message-row you-message"
    } else {
        messageRow.className = "message-row other-message"
    }
    let messageText = document.createElement("div")
    messageText.className = "message-text"
    messageText.innerHTML = content

    let messageOrigin = document.createElement("div")
    messageOrigin.className = "message-origin"
    messageOrigin.innerHTML = origin

    messageRow.appendChild(messageText)
    messageRow.appendChild(messageOrigin)

    messageList.insertAdjacentElement('afterbegin', messageRow)

}

function buttonClickNewNode() {
    let inputText = document.getElementById("newNode")
    let newAddress = inputText.value
    if (newAddress != "" && checkValidIPv4(newAddress)) {
        $.ajax({
            type: "POST",
            url: "http://localhost:8080/node",
            data: { "value": newAddress },
            success: () => {
                inputText.value = ""
            },
            error: (status) => {
                if (status.status == 400) {
                    alert("You cannot add your own gossiper!")
                }
            }
        })
    } else {
        alert(`${newAddress} has not a valid format`)
    }

    inputText.value = ""
}


function checkValidIPv4(address) {
    ipPort = address.split(":")
    if (ipPort.length > 2) {
        return false
    }
    port = Number(ipPort[1])
    let regex = /^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/
    return regex.test(ipPort[0]) && port >= 0 && port <= 65535
}

function checkValidSha256(hash) {
    let regex = /^([a-f0-9]{64})$/
    return regex.test(hash)
}

setInterval(() => {
    getMessages()
    getAllNodes()
    getAllOrigins()
    getMatches()
}, 1000)

function getMessages() {
    $.ajax({
        type: "GET",
        url: "http://localhost:8080/message",
        data: {
            "name": active
        },
        dataType: 'json',
        success: function (data, status) {
            let list = document.getElementById("chat-message-list")

            while (list.hasChildNodes()) {
                list.removeChild(list.lastChild)
            }

            for (let msg of data) {
                addNewMessage(msg.Origin, msg.Text)
            }
        }, error: status => {
            let list = document.getElementById("chat-message-list")

            while (list.hasChildNodes()) {
                list.removeChild(list.lastChild)
            }
        }
    })
}

function getAllNodes() {
    $.ajax({
        type: "GET",
        url: "http://localhost:8080/node",
        dataType: 'json',
        success: function (data, status) {
            let list = document.getElementById("node-list")

            while (list.hasChildNodes()) {
                list.removeChild(list.lastChild)
            }

            for (let peerAddr of data.sort()) {
                let addrDiv = document.createElement("div")
                addrDiv.className = "node"
                addrDiv.innerHTML = `${peerAddr}`
                list.appendChild(addrDiv)
            }
        }
    })
}

function getMatches() {
    $.ajax({
        type: "GET",
        url: "http://localhost:8080/matches",
        dataType: 'json',
        success: function (data, status) {
            let list = document.getElementById("search-list")

            while (list.hasChildNodes()) {
                list.removeChild(list.lastChild)
            }

            for (let match of data) {
                fileName = match["FileName"]
                origin = match["Origin"]
                metahash = match["MetaHash"]
                let matchDiv = document.createElement("div")
                matchDiv.className = "search"
                matchDiv.onclick =(function() {
                    var currentFileName = fileName;
                    var currentMetaHash = metahash;
                    var currentOrigin = origin
                    return function() { 
                        download(currentOrigin, currentMetaHash, currentFileName)
                    }
                })()
                let nameP = document.createElement("p")
                nameP.id = "name"
                nameP.innerHTML = fileName
                matchDiv.appendChild(nameP)

                let originP = document.createElement("p")
                originP.id = "origin"
                originP.innerHTML = origin
                matchDiv.appendChild(originP)

                let metahashP = document.createElement("p")
                metahashP.id = "origin"
                metahashP.innerHTML = metahash
                metahashP.hidden = true
                matchDiv.appendChild(metahashP)
                
                list.appendChild(matchDiv)
            }
        }
    })
}

function buttonClickSearch() {
    let keywords = document.getElementById("keywords").value.split(",")
    keywords = keywords.filter(x => x != "")
        if (keywords.length > 0) {
        $.ajax({
            type: "POST",
            url: "http://localhost:8080/search", 
            data: {
                "keywords": keywords.join()
            },
            success: () => {
                console.log(keywords)
                document.getElementById("keywords").value = ""
            }, error: (status) => {
                console.log(status)
                document.getElementById("keywords").value = ""
            }
        })
    }else{
        document.getElementById("keywords").value = ""
    }

}

function buttonClickDownload() {

    if (active === "Rumors") {
        alert("You cannot download from Rumors")
        return
    }

    let metaHashInput = document.getElementById("newHash")
    let [fileName, metaHash] = metaHashInput.value.split(";")
    download(active, metaHash, fileName)
    metaHashInput.value = ""

}

function download(from, metaHash, fileName){
    if (checkValidSha256(metaHash)) {
        $.ajax({
            type: "POST",
            url: "http://localhost:8080/download",
            data: {
                "metahash": metaHash,
                "from": from,
                "fileName": fileName
            },error: (status) => {
                console.log(status)
            }
        })
    } else {
        alert("Invalid SHA256")
    }
}


function fileSelectionHandler(e) {

    let file = e.target.files[0]
    $.ajax({
        type: "POST",
        url: "http://localhost:8080/shareFile",
        data: { "fileName": file.name },
    })

}

function getAllOrigins() {
    $.ajax({
        type: "GET",
        url: "http://localhost:8080/origin",
        dataType: 'json',
        success: function (data, status) {
            let list = document.getElementById("conversation-list")
            while (list.hasChildNodes()) {
                list.removeChild(list.lastChild)
            }

            addConversation("Rumors")

            for (let origin of data.sort()) {
                addConversation(origin)
            }
        }
    })
}

function changeActive(name) {
    let convList = document.getElementById("conversation-list")

    for (let ch of convList.childNodes) {
        if (ch.className === "conversation active") {
            ch.className = "conversation"
        } else {
            let title = ch.lastChild
            if (title.innerHTML === name) {
                ch.className = "conversation active"
            }
        }

    }

}


function addConversation(name) {
    let convList = document.getElementById("conversation-list")
    let cl = "conversation"
    if (name === active) {
        cl = "conversation active"
    }

    let convDiv = document.createElement("div")
    convDiv.className = cl
    convDiv.onclick = () => {
        active = name
        changeActive(name)
        getMessages()
        document.getElementById("span-name").innerHTML = name

    }

    let titleDiv = document.createElement("div")
    titleDiv.className = "title-text"
    titleDiv.innerHTML = name

    convDiv.appendChild(titleDiv)
    convList.appendChild(convDiv)

}



$.ajax({
    type: "GET",
    url: "http://localhost:8080/id",
    success: function (data, status, xhr) {
        let name = JSON.parse(data);
        myID = name.toString()
        document.getElementById("nodeName").innerHTML = myID.substring(0, 12)
    }
});

getMessages()
getAllNodes()
getAllOrigins()