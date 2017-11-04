$(document).ready(function () {

    $.get("/devices", function (data) {
        var obj = JSON.parse(data);
        var data_length = obj.length;

        $("#result").append('<div id="container" class="container">');
        $("#container").append('<div id="row" class="row">');

        while (data_length > 0) {
            var info_card = "info-card" + data_length;
            var front = "front" + data_length;
            var back = "back" + data_length;

            // Front Side
            $("#row").append('<div id="' + info_card + '" class="info-card">');
            $("#" + info_card).append('<div id="' + front + '"class="front">');
            $('#' + front).append('<img class="card-image" src="img/fridge.ico" />');
            $('#' + front).append('</div'); // Close front

            // Back side
            $("#" + info_card).append('<div id="' + back + '"class="back">');

            // Device Info
            $("#" + back).append('<p> Type: ' + obj[data_length - 1]["meta"]["type"] + '</p>');
            $("#" + back).append('<p>Name: ' + obj[data_length - 1]["meta"]["name"] + '</p>');

            var device_data = obj[data_length - 1]["data"];

            var dateAndValueCam1 = device_data.TopCompart[device_data.TopCompart.length - 1].split(':');
            var dateAndValueCam2 = device_data.BotCompart[device_data.BotCompart.length - 1].split(':');

            $("#" + back).append('<p>' +
                "Cam1Time: " + new Date(parseInt(dateAndValueCam1[0])).toLocaleString() + '<br>' +
                "Cam1Temp: " + dateAndValueCam1[1] + '<br>' + '<br>' +
                "Cam2Time: " + new Date(parseInt(dateAndValueCam2[0])).toLocaleString() + '<br>' +
                "Cam2Temp: " + dateAndValueCam2[1] +
                '</p>');

            // Button get detailed data
            $("#"+back).append('<button type="button" class="btn btn-basic" id="dataBtn' + data_length + '">' + 'Detailed data' + '</button>');
            $("#"+ "dataBtn" + data_length).on('click', function () {
                var id = this.id.replace( /^\D+/g, '');
                window.location = "fridge.html?type=" + obj[id - 1]["meta"]["type"] + "&name="
                    + obj[id - 1]["meta"]["name"] + "&mac=" +  obj[id - 1]["meta"]["mac"];
            });

            $('#' + info_card).append('</div'); // Close
            data_length--;
        }
    });
});