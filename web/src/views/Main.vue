<template lang="pug">
div
  .active-device-list.px-3.py-2(
    :class="{ 'no-active-devices': !playbackDevice }"
  )
    div(v-if="playbackDevicesInitiallyRequested")
      div(v-if="playbackDevice")
        i.fa.fa-volume-up.mr-2
        | Currently playing on "{{ playbackDevice.name }}".
      .center-sm(v-else)
        b-button#fetch-active-devices-btn.btn-lg(
          variant="warning",
          @click="fetchActiveDevices()"
        ) No playback on any device. Click&nbsp;to&nbsp;refresh.

  .container
    .row.mt-4
      .slot-card.col-lg-4.col-md-6(v-for="(state, slotNumber) in playerStates")
        .card.mb-4.bg-light.box-shadow
          img.card-img-top(
            :src="state.albumArtLargeURL",
            alt="Album art provided by Spotify"
          )
          b-progress(:max="state.totalTracks", variant="success")
            b-progress-bar(:value="state.trackIndex")
          .card-body
            .card-content
              h5.card-title {{ state.trackName }}
              .info-table
                .table-row(v-if="state.playlistName")
                  .table-cell
                    i.fa.fa-list-ul
                  .table-cell
                    p {{ state.playlistName }}
                .table-row
                  .table-cell
                    i.fa.fa-music
                  .table-cell
                    p {{ state.albumName }}
                .table-row
                  .table-cell
                    i.fa.fa-user
                  .table-cell
                    p {{ state.artistName }}
                .table-row
                  .table-cell
                    i.fa.fa-hourglass-end
                  .table-cell
                    p {{ state.progress | time }} / {{ state.duration | time }} (track {{ state.trackIndex }} of {{ state.totalTracks }})
                .table-row
                  .table-cell
                    i.fa.fa-spotify
                  .table-cell
                    p
                      a(:href="state.linkToContext")
                        | Open in Spotify
                        i.fa.fa-external-link.ml-1
            .row.mt-2
              .col.p-1
                b-button.overwrite-btn.btn-block(
                  @click="updatePlayerState(slotNumber)",
                  :disabled="!playbackDevice",
                  variant="primary"
                )
                  i.fa.fa-stop-circle.fa-lg
              .col.p-1
                template(v-if="activeDevices.length > 1")
                  b-dropdown.resume-btn.btn-block(
                    split,
                    @click="restoreFromPlayerState(slotNumber)",
                    variant="success"
                  )
                    template(#button-content)
                      i.fa.fa-play-circle.fa-lg.col
                    b-dropdown-item.disabled Start playback on:
                    b-dropdown-divider
                    b-dropdown-item(
                      v-for="device in activeDevices",
                      @click="restoreFromPlayerState(slotNumber, device.id, device.name)",
                      :key="device.id"
                    ) {{ device.name }}
                template(v-else)
                  b-button.resume-btn.btn-block(
                    @click="restoreFromPlayerState(slotNumber)",
                    variant="success"
                  )
                    i.fa.fa-play-circle.fa-lg
              .col.p-1
                b-button.delete-btn.btn-block(
                  @click="deletePlayerState(slotNumber)",
                  variant="danger"
                )
                  i.fa.fa-trash.fa-lg
      a#suspend-btn.floating-btn(
        @click="storePlayerState",
        :disabled="!playbackDevice"
      )
        i.fa.fa-pause-circle
  div
</template>

<script>
import intro from "../lib/intro"

export default {
  name: "Main",
  props: [
    "firstRun" // gets set by the router resp. the consent view
  ],
  data: function () {
    return {
      playerStates: [],
      playbackDevicesInitiallyRequested: false,
      playbackDevice: undefined,
      activeDevices: [],
      showModal: false,
      errorMessage: ""
    }
  },
  filters: {
    time: function (millis) {
      const inSecs = Math.round(millis / 1000)
      const hours = Math.floor(inSecs / 3600)
      const remaining = inSecs - hours * 3600
      const minutes = Math.floor(remaining / 60)
      const seconds = remaining - minutes * 60

      return `${(hours > 0) ? hours + ":" : ""}${(minutes < 10) ? "0" : ""}${minutes}:${(seconds < 10) ? "0" : ""}${seconds}`
    }
  },
  methods: {
    showErrorMessage: function (msg) {
      this.$bvModal.msgBoxOk("Oh no! " + msg)
    },
    fetchActiveDevices: function () {
      return this.$api.fetchActiveDevices().then((activeDevices) => {
        this.activeDevices = activeDevices
        this.playbackDevice = undefined
        this.playbackDevicesInitiallyRequested = true

        activeDevices.forEach(device => {
          if (device.active) {
            this.playbackDevice = device
          }
        });

        // It is save to call this every time - if the refresh button is shown triggering next is necessary,
        // if it is not shown this code will not be called.
        // If the intro is not shown, calling next() has no effect.
        if (this.playbackDevice) {
          intro.next()
        }
      }, (err) => {
        this.playbackDevice = undefined
        this.playbackDevicesInitiallyRequested = true

        this.showErrorMessage("Failed to request active devices. This should not happen. Please try again.")
        console.error("Failed to request actives devices from backend.", err)
      })
    },
    fetchPlayerStates: function (fetchActiveDevicesPromise) {
      return this.$api.fetchPlayerStates().then(async (playerStates) => {
        this.playerStates = playerStates.reverse()

        // TODO: if (this.playerStates.length === 0 && this.firstRun) {
        if (this.firstRun) {
          console.debug("This seems to be the first run of Cassette. Running the intro.")

          if (fetchActiveDevicesPromise) {
            await fetchActiveDevicesPromise
          }

          const activeDevicePresent = this.playbackDevice !== undefined
          intro.start(activeDevicePresent)
        }
      }, (err) => {
        this.showErrorMessage("Failed to request your player states. This should not happen. Please try again.")
        console.error("Failed to request player states from backend.", err)
      })
    },
    updatePlayerState: function (idx) {
      const slotNumber = this.playerStates.length - 1 - idx

      this.$api.updatePlayerState(slotNumber).then(async () => {
        console.info(`Successfully updated player state in slot ${slotNumber}.`)

        await this.fetchPlayerStates()

        intro.next()
      }, (err) => {
        this.showErrorMessage("Failed to update player state. This should not happen. Please try again.")
        console.error(`Failed to update player state in slot ${slotNumber}.`, err)
      })
    },
    storePlayerState: function () {
      this.$api.storePlayerState().then(async () => {
        console.info("Successfully updated player state in new slot.")

        await this.fetchPlayerStates()

        console.info("States fetched")

        // We have to ensure the DOM element actually exists before progressing the tour
        this.$nextTick(() => {
          console.log(document.querySelector(".slot-card:first-of-type"))
          intro.next()
        })
      }, (err) => {
        this.showErrorMessage("Failed to store new player state. This should not happen. Please try again.")
        console.error("Failed to store new player state.", err)
      })
    },
    deletePlayerState: async function (idx) {
      const slotNumber = this.playerStates.length - 1 - idx

      const ok = await this.$bvModal.msgBoxConfirm("Are you sure you want to delete this state? This cannot be undone.", {
        okVariant: "danger",
        okTitle: "Delete"
      })

      if (!ok) {
        return
      }

      this.$api.deletePlayerState(slotNumber).then(() => {
        console.info(`Successfully deleted player state in slot ${slotNumber}.`)

        // Avoid re-fetching the player states, we can compute the change locally
        // The playerStates array is reversed - keep this in mind!
        this.playerStates.splice(idx, 1);
      }, (err) => {
        this.showErrorMessage("Failed to delete the player state. This should not happen. Please try again.")
        console.error(`Failed to delete player state in slot ${slotNumber}.`, err)
      })
    },
    restoreFromPlayerState: function (idx, deviceID, deviceName) {
      const slotNumber = this.playerStates.length - 1 - idx

      this.$api.restoreFromPlayerState(slotNumber, deviceID).then(() => {
        console.info(`Successfully restored player state from slot ${slotNumber} on device ${deviceID}.`)

        intro.next()
      }, (err) => {
        this.showErrorMessage(`Failed to restore player state on ${(deviceName !== undefined) ? `"${deviceName}"` : "currently active device"}.
        Please make sure Spotify is active on this device. This can be done by starting some arbitrary track. Please try again then.
        If the issue persists there might also be an issue with the specific track.`)
        console.error(`Failed to restore player state from slot ${slotNumber} on device ${deviceID}.`, err)
      })
    }
  },
  mounted: function () {
    this.$api.fetchCSRFToken().then((csrfToken) => {
      console.info("Successfully fetched CSRF token.")

      this.$api.setCSRFToken(csrfToken)

      this.fetchPlayerStates(this.fetchActiveDevices())
    }, (err) => {
      this.showErrorMessage("Failed initializing the app. Please reload the page.")
      console.error("Failed fetching the CSRF token.", err)
    })
  }
}
</script>