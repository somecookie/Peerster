let numberMessage = 0

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


function addNewMessageToTable(origin, content) {
    let table = document.getElementById("chatBox")
    let row = table.insertRow(table.rows.length)
    let from = row.insertCell(0)
    let message = row.insertCell(1)

    from.innerHTML = origin

    let paragraph = document.createElement("p")
    let node = document.createTextNode(content)
    paragraph.appendChild(node)

    message.appendChild(paragraph)
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
        alert("Wrong format for the address")
    }

    inputText.value = ""
}

function addNewNodeToList(address) {
    let node = document.createElement("LI")
    let textNode = document.createTextNode(address)
    node.appendChild(textNode)
    document.getElementById("nodes").appendChild(node)
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
                addNewMessageToTable(message.Origin, message.Text)
            }

            getAllNodes()
            getAllOrigins()
        }
    })
}, 1000)


function getAllNodes() {
    $.ajax({
        type: "GET",
        url: "http://localhost:8080/node",
        dataType: 'json',
        success: function (data, status) {
            let list = document.getElementById("nodes")
            while (list.hasChildNodes()) {
                list.removeChild(list.lastChild)
            }

            for (let peerAddr of data.sort()) {
                addNewNodeToList(peerAddr)
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
        document.getElementById("nodeName").innerHTML = name.toString()
    }
});
