/*
 *  Isaac Race Server stuff
 */

// Home - "Learn More" button
$('#learn-more').click(function() {
    $('html, body').animate({
        scrollTop: $("#main").offset().top
    }, 1000);
});

// Leaderboards - Leaderboard buttons
var activeLeaderboard = 'seeded';
var transition = false;
function showLeaderboard(type) {
    // Header buttons
    $('#leaderboard-seeded-button').addClass('inactive');
    $('#leaderboard-unseeded-button').addClass('inactive');
    $('#leaderboard-other-button').addClass('inactive');
    $('#leaderboard-' + type + '-button').removeClass('inactive');

    // Fade out the old leaderboard and fade in the new one
    transition = true;
    $('#leaderboard-' + activeLeaderboard).fadeOut(350, function() {
        $('#leaderboard-' + type).fadeIn(350, function() {
            activeLeaderboard = type;
            transition = false;
        });
    });
}
$('#leaderboard-seeded-button').click(function() {
    if (activeLeaderboard !== 'seeded' && transition === false) {
        showLeaderboard('seeded');
    }
});
$('#leaderboard-unseeded-button').click(function() {
    if (activeLeaderboard !== 'unseeded' && transition === false) {
        showLeaderboard('unseeded');
    }
});
$('#leaderboard-other-button').click(function() {
    if (activeLeaderboard !== 'other' && transition === false) {
        showLeaderboard('other');
    }
});

/*
    Alpha by HTML5 UP
    html5up.net | @ajlkn
    Free for personal and commercial use under the CCA 3.0 license (html5up.net/license)
*/
(function($) {

    skel.breakpoints({
        wide: '(max-width: 1680px)',
        normal: '(max-width: 1280px)',
        narrow: '(max-width: 980px)',
        narrower: '(max-width: 840px)',
        mobile: '(max-width: 736px)',
        mobilep: '(max-width: 480px)'
    });

    $(function() {

        var    $window = $(window),
            $body = $('body'),
            $header = $('#header'),
            $banner = $('#banner');

        // Fix: Placeholder polyfill.
            $('form').placeholder();

        // Prioritize "important" elements on narrower.
            skel.on('+narrower -narrower', function() {
                $.prioritize(
                    '.important\\28 narrower\\29',
                    skel.breakpoint('narrower').active
                );
            });

        // Dropdowns.
            $('#nav > ul').dropotron({
                alignment: 'right'
            });

        // Off-Canvas Navigation.

            // Navigation Button.
                $(
                    '<div id="navButton">' +
                        '<a href="#navPanel" class="toggle"></a>' +
                    '</div>'
                )
                    .appendTo($body);

            // Navigation Panel.
                $(
                    '<div id="navPanel">' +
                        '<nav>' +
                            $('#nav').navList() +
                        '</nav>' +
                    '</div>'
                )
                    .appendTo($body)
                    .panel({
                        delay: 500,
                        hideOnClick: true,
                        hideOnSwipe: true,
                        resetScroll: true,
                        resetForms: true,
                        side: 'left',
                        target: $body,
                        visibleClass: 'navPanel-visible'
                    });

            // Fix: Remove navPanel transitions on WP<10 (poor/buggy performance).
                if (skel.vars.os == 'wp' && skel.vars.osVersion < 10)
                    $('#navButton, #navPanel, #page-wrapper')
                        .css('transition', 'none');

        // Header.
        // If the header is using "alt" styling and #banner is present, use scrollwatch
        // to revert it back to normal styling once the user scrolls past the banner.
        // Note: This is disabled on mobile devices.
            if (!skel.vars.mobile &&
                $header.hasClass('alt') &&
                $banner.length > 0) {

                $window.on('load', function() {

                    $banner.scrollwatch({
                        delay:        0,
                        range:        0.5,
                        anchor:        'top',
                        on:            function() { $header.addClass('alt reveal'); },
                        off:        function() { $header.removeClass('alt'); }
                    });

                });

            }

    });

})(jQuery);
