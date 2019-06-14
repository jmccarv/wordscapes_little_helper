var prevSearch = { letters: "", template: "" };
function addBox() {
    var t = $("span.template");
    if (t.children().length >= 7)
        return;
    var box = $("<textarea>", { cols: 3,
        rows: 1,
        maxLength: 1,
        "class": "mx-1"
    });
    box.change(doSearch);
    box.keyup(doSearch);
    box.click(function () { $(this).select(); });
    box.appendTo(t);
    t.children().val("");
    doSearch();
}
function clearForm() {
    $("input.letters").val("");
    $("span.template").children().val("");
    $("div.results").empty();
}
function removeBox() {
    var t = $("span.template");
    if (t.children().length <= 3)
        return;
    t.children().last().remove();
    t.children().val("");
    doSearch();
}
var searchID = 0;
function searchDone() {
    var myID = ++searchID;
    return function (data) {
        // Ignore results unless they're from the most recent query
        if (myID != searchID) {
            //console.log("Ignoring old query results "+myID+"  current="+searchID)
            return;
        }
        $("div.results").empty();
        data.forEach(function (value) {
            $("<div class=\"col\">" + value + "</div>").appendTo("div.results");
        });
    };
}
function doSearch() {
    var letters = $("input.letters").val().toString().toLowerCase();
    var template = $("span.template").children().toArray().map(function (e) {
        return e.value.length ? e.value : '.';
    }).join('').toLowerCase();
    $("span.template-count").text(template.length);
    if (prevSearch.letters === letters && prevSearch.template === template)
        return;
    prevSearch.letters = letters;
    prevSearch.template = template;
    $("div.results").empty();
    if (letters.length < template.length) {
        $("input.letters").addClass("bg-warning");
        return;
    }
    $("input.letters").removeClass("bg-warning");
    $.ajax({
        url: window.location.href + "/api/search",
        data: { letters: letters, template: template },
        type: "GET",
        dataType: "json"
    })
        .done(searchDone())
        .fail(function (xhr, status, errorThrown) {
        console.log("Error: ${errorThrown}");
    });
}
$(document).ready(function () {
    $("input.letters").change(doSearch);
    $("input.letters").keyup(doSearch);
    for (var i = 0; i < 4; i++) {
        addBox();
    }
});
