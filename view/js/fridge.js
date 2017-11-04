function parseURLParams(url) {
    var queryStart = url.indexOf("?") + 1,
        queryEnd = url.indexOf("#") + 1 || url.length + 1,
        query = url.slice(queryStart, queryEnd - 1),
        pairs = query.replace(/\+/g, " ").split("&"),
        parms = {}, i, n, v, nv;

    if (query === url || query === "") {
        return;
    }

    for (i = 0; i < pairs.length; i++) {
        nv = pairs[i].split("=", 2);
        n = decodeURIComponent(nv[0]);
        v = decodeURIComponent(nv[1]);

        if (!parms.hasOwnProperty(n)) {
            parms[n] = [];
        }

        parms[n].push(nv.length === 2 ? v : null);
    }
    return parms;
}

function setRequestParams(id, xhr) {
    var url = "/devices/" + id + "/config?mac=" + urlParams["mac"] + "&type=" + urlParams["type"] + "&name=" + urlParams["name"];
    xhr.open("PATCH", url, true);
    xhr.setRequestHeader("Content-type", "application/json");
    xhr.onreadystatechange = function () {
        if (xhr.readyState === 4 && xhr.status === 200) {
            alert("Data have been delivered!");
            console.log(typeof xhr.responseText)
        } else if (xhr.readyState === 4 && xhr.status === 400) {
            alert(xhr.responseText);
        }
    };
}

function sendDevConfigFreq(id, collectFreq, sendFreq) {
    var xhr = new XMLHttpRequest();
    setRequestParams(id, xhr);

    var config = JSON.stringify(
        {
            "mac": id[0],
            "data": {
                "collectFreq": collectFreq,
                "sendFreq": sendFreq
            }
        });

    xhr.send(config);
}

function sendDevConfigTurnedOn(id, turnedOn) {
    var xhr = new XMLHttpRequest();
    setRequestParams(id, xhr);

    var config = JSON.stringify(
        {
            "mac": id[0],
            "data": {
                "turnedOn": turnedOn
            }
        });

    xhr.send(config);
}

function sendDevDataStreamOn(id, streamOn) {
    var xhr = new XMLHttpRequest();
    setRequestParams(id, xhr);

    var config = JSON.stringify(
        {
            "mac": id[0],
            "data": {
                "streamOn": streamOn
            }
        });

    xhr.send(config);
}

function setDevDataFields(obj) {
    document.getElementById('devType').value = obj["meta"]["type"];
    document.getElementById('devName').value = obj["meta"]["name"];
}

function setDevConfigFields(obj) {
    if (obj["turnedOn"]) {
        document.getElementById('turnedOnBtn').innerHTML = "On";
        document.getElementById('turnedOnBtn').className = "btn btn-success";
    } else {
        document.getElementById('turnedOnBtn').innerHTML = "Off";
        document.getElementById('turnedOnBtn').className = "btn btn-danger";
    }

    document.getElementById('collectFreq').value = obj["collectFreq"];
    document.getElementById('sendFreq').value = obj["sendFreq"];

    if (obj["streamOn"]) {
        document.getElementById('streamOnBtn').innerHTML = "On";
        document.getElementById('streamOnBtn').className = "btn btn-success";
    } else {
        document.getElementById('streamOnBtn').innerHTML = "Off";
        document.getElementById('streamOnBtn').className = "btn btn-danger";
    }
}

function printFridgeChart(obj) {
    Highcharts.setOptions({
        global: {
            useUTC: false
        }
    });

    Highcharts.stockChart('container', {
        chart: {
            events: {
                load: function () {
                    var seriesTopCompart = this.series[0];
                    var seriesBotCompart = this.series[1];
                    var timerForRepaint = 50;
                    var repaint = function (fridge) {
                        for (key in fridge.data.BotCompart) {
                            var x = parseInt(key);
                            var y = parseFloat(fridge.data.BotCompart[key]);
                            seriesBotCompart.addPoint([x, y], true, true);
                        }
                        for (key in fridge.data.TopCompart) {
                            var x = parseInt(key);
                            var y = parseFloat(fridge.data.TopCompart[key]);
                            seriesTopCompart.addPoint([x, y], true, true);
                        }
                    };

                    var timerId = setInterval(function () {
                        if (showStreamData === true) {
                            var fridgeData = fridgeDataList.shift()
                            if (fridgeData !== undefined) {
                                repaint(fridgeData)
                            }
                        }
                    }, timerForRepaint)
                }
            }
        },

        rangeSelector: {
            buttons: [{
                count: 1,
                type: 'minute',
                text: '1M'
            }, {
                count: 5,
                type: 'minute',
                text: '5M'
            }, {
                type: 'all',
                text: 'All'
            }],
            inputEnabled: false,
            selected: 0
        },

        exporting: {
            enabled: false
        },

        series: [{
            name: 'TopCompart',
            data: (function () {
                var data = [];
                for (var i = 0; i < obj["data"]["TopCompart"].length; ++i) {
                    data.push({
                        x: parseInt(obj["data"]["TopCompart"][i].split(':')[0]),
                        y: parseFloat(obj["data"]["TopCompart"][i].split(':')[1])
                    });
                }
                return data;
            }())
        }, {
            name: 'BotCompart',
            data: (function () {
                var data = [];
                for (var i = 0; i < obj["data"]["BotCompart"].length; ++i) {
                    data.push({
                        x: parseInt(obj["data"]["BotCompart"][i].split(':')[0]),
                        y: parseFloat(obj["data"]["BotCompart"][i].split(':')[1])
                    });
                }
                return data;
            }())
        }]
    })
}

var url = window.location.href.split("/");
var urlParams = parseURLParams(window.location.href);
var domain = url[2].split(":");

var showStreamData = false;
var fridgeDataList = [];
var webSocket = new WebSocket("ws://" + domain[0] + ":3546" + "/devices/" + String(urlParams["mac"]));
webSocket.onmessage = function (event) {
    var incomingMessage = event.data;
    var fridgeData = JSON.parse(incomingMessage);
    fridgeDataList.push(fridgeData);
};

$(document).ready(function () {
    var urlParams = parseURLParams(window.location.href);

    $.get("/devices/id/data" + "?mac=" + urlParams["mac"] + "&type=" + urlParams["type"] + "&name=" + urlParams["name"], function (data) {
        var obj = JSON.parse(data);
        setDevDataFields(obj);
        printFridgeChart(obj);
    });

    $.get("/devices/" + urlParams["mac"] + "/config?mac=" + urlParams["mac"] + "&type=" + urlParams["type"] + "&name=" + urlParams["name"], function (data) {
        var obj = JSON.parse(data);
        setDevConfigFields(obj);
    });

    document.getElementById("turnedOnBtn").onclick = function () {
        var value = this.innerHTML;
        if (value === "On") {
            sendDevConfigTurnedOn(
                urlParams["mac"],
                false
            );
            this.innerHTML = "Off";
            this.className = "btn btn-danger";
        } else {
            sendDevConfigTurnedOn(
                urlParams["mac"],
                true
            );
            this.innerHTML = "On";
            this.className = "btn btn-success";
        }
    };

    document.getElementById("updateBtn").onclick = function () {
        sendDevConfigFreq(
            urlParams["mac"],
            parseInt(document.getElementById('collectFreq').value),
            parseInt(document.getElementById('sendFreq').value)
        );
    };

    document.getElementById("streamOnBtn").onclick = function () {
        var value = this.innerHTML;
        if (value === "On") {
            /*sendDevDataStreamOn(
                urlParams["mac"],
                false
            );*/
            showStreamData = false;
            this.innerHTML = "Off";
            this.className = "btn btn-danger";
        } else {
            /*sendDevDataStreamOn(
                urlParams["mac"],
                true
            );*/
            showStreamData = true;
            this.innerHTML = "On";
            this.className = "btn btn-success";
        }
    };
});
