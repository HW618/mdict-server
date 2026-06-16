$(document).ready(function () {
    $(".more_example_title").click(function () {
        var exampleSiblings = $(this).siblings("p.en_example")
        if (exampleSiblings.length > 0) {
            if (exampleSiblings.hasClass("active")) {
                exampleSiblings.removeClass("active");
                exampleSiblings.css({
                    "display": "none"
                });
            } else {
                exampleSiblings.addClass("active");
                exampleSiblings.css({
                    "display": "block"
                });
            }
        }
        if (exampleSiblings.length <= 0) {
            $(this).css({
                "color": "red"
            });
        }
    });
});