let numberMessage = 0
let myID = ""
let active = "General"

function buttonClickNewMessage() {
    let inputText = document.getElementById("newMessage")
    

    if (inputText.value != "") {
        $.ajax({
            type: "POST",
            url: "http://localhost:8080/message",
            data: { "value": inputText.value},
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
    if(origin === myID){
        messageRow.className = "message-row you-message"
    }else{
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
                if(status.status == 400){
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

setInterval(() => {
    $.ajax({
        type: "GET",
        url: "http://localhost:8080/message",
        dataType: 'json',
        success: function (data, status) {
            for (let i = numberMessage; i < data.length; i++) {
                numberMessage++
                let message = data[i]
                addNewMessage(message.Origin, message.Text)

            }

            getAllNodes()
            //getAllOrigins()
        }
    })
}, 1000)


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

function addNewOriginToList(origin) {
    let node = document.createElement("LI")
    let textNode = document.createTextNode(origin)
    node.appendChild(textNode)
    document.getElementById("origins").appendChild(node)


}

function getAllOrigins() {
    $.ajax({
        type: "GET",
        url: "http://localhost:8080/origin",
        dataType: 'json',
        success: function (data, status) {
            let list = document.getElementById("origins")
            while (list.hasChildNodes()) {
                list.removeChild(list.lastChild)
            }

            for (let origin of data.sort()) {
                addNewOriginToList(origin)
            }
        }
    })
}



getAllNodes()
getAllOrigins()


$.ajax({
    type: "GET",
    url: "http://localhost:8080/id",
    success: function (data, status, xhr) {
        let name = JSON.parse(data);
        myID = name.toString()
        document.getElementById("nodeName").innerHTML = myID.substring(0,12)
    }
});
