import Shepherd from "shepherd.js"

const Intro = function () {
    const nextBtn = {
        action: () => {
            this.next()
        },
        classes: "btn btn-primary",
        text: "Next",
    }

    const tour = new Shepherd.Tour({
        defaultStepOptions: {
            scrollTo: {
                behavior: "smooth",
                block: "center",
            },
            cancelIcon: {
                enabled: true,
            },
            arrow: true,
            buttons: [],
        },
        useModalOverlay: true,
    })

    this.start = (activeDevicePresent) => {
        // Do not start the tour several times
        if (tour.isActive()) {
            return
        }

        tour.addStep({
            title: "Welcome to Cassette! &#128526",
            text: "Although Cassette is supposed to be easy to use this short introduction will give you some hints on how to use this tool. Feel free to skip if you prefer to find out yourself &#128521.<hr>",
            buttons: [
                {
                    action: () => {
                        tour.cancel()
                    },
                    classes: "btn btn-warning",
                    text: "Skip intro",
                },
                nextBtn,
            ],
        })

        if (!activeDevicePresent) {
            tour.addStep({
                attachTo: {
                    element: "#fetch-active-devices-btn",
                    on: "auto",
                },
                title: "Refresh active devices",
                text: "It seems like there is no active device currently. Please make sure Spotify is currently playing (and not in offline mode). Sometimes the mobile app might require you to pause/unpause/skip in order to synchronize with Spotify's platform. This sometimes is a little buggy - and nothing Cassette could do about it. &#128533<hr>In order to continue with the tour please click the big, yellow button.",
            })
        }

        tour.addStep({
            attachTo: {
                element: "#suspend-btn",
                on: "auto",
            },
            title: "Suspend your current state",
            text: "Pause and store your current state in a new 'slot'. You can have as many slots as you want - allowing you to suspend an arbitrary amount of audiobooks/albums/playlists.<hr>In order to continue with the tour please make sure there is still some active playback and click the button.",
        })

        tour.addStep({
            attachTo: {
                element: ".slot-card:first-of-type",
                on: "auto",
            },
            title: "Your first slot/state",
            text: "We call this a slot, a place to store Spotify's player state into. The state stored in a slot can be resumed or overwritten. Of course you might also remove the slot.<hr>",
            buttons: [nextBtn],
        })

        tour.addStep({
            attachTo: {
                element: ".slot-card:first-of-type .progress",
                on: "auto",
            },
            title: "Your first slot/state",
            text: "This bar indicates your progress within the current album/playlist. Especially handy with audiobooks. Don't worry if this looks just grey - just keep on listening.<hr>",
            buttons: [nextBtn],
        })

        tour.addStep({
            attachTo: {
                element: ".slot-card:first-of-type .resume-btn",
                on: "auto",
            },
            title: "Restore a slot/resume a state",
            text: "Click here to restore this state and continue playback on the currently active device. If there are several active devices you may use the dropdown button to select a specific device. Don't be surprised: Cassette jumps back 10s in the track in order to ease getting back into audiobooks etc.<hr>In order to continue with the tour please make sure there is an active device and click the button. Due to energy saving measures a mobile device running a Spotify app might not show up in your active devices unless the app is running in the foreground or actually playing some music.",
        })

        tour.addStep({
            attachTo: {
                element: ".slot-card:first-of-type .overwrite-btn",
                on: "auto",
            },
            title: "Overwrite a slot/update a state",
            text: "Click to update the state stored in this slot.<hr>In order to continue with the tour please click the button.",
        })

        tour.addStep({
            attachTo: {
                element: ".slot-card:first-of-type .delete-btn",
                on: "auto",
            },
            title: "Remove a slot",
            text: "We reached the end of this interactive tour, feel free to remove this test slot. Have fun using Cassette! &#128513<hr>",
            buttons: [
                {
                    action: () => {
                        tour.complete()
                    },
                    classes: "btn btn-success",
                    text: "Ok, let's go!",
                },
            ],
        })

        tour.start()
    }

    this.next = () => {
        if (tour.isActive()) {
            console.debug("Progressing tour...")
            tour.next()
        }
    }
}

export default new Intro()
