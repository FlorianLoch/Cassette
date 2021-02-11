<template lang="pug">
div
  .active-device-list.px-3.py-2(
    :class="{ 'no-active-devices': !playbackDevice }"
  )
    div(v-if="playbackDevicesInitiallyRequested")
      div(v-if="playbackDevice")
        i.fas.fa-volume-up.mr-2
        | Currently playing on "{{ playbackDevice.name }}".
      .center-sm(v-else)
        b-button.btn-lg(variant="warning", @click="fetchActiveDevices()") No playback on any device. Click&nbsp;to&nbsp;refresh.

  .container
    .row.mt-4
      .col-lg-4.col-md-6(v-for="(state, slotNumber) in playerStates")
        .card.mb-4.bg-light.box-shadow
          img.card-img-top(
            :src="state.albumArtLargeURL",
            alt="Album art provided by Spotify"
          )
          b-progress(:max="state.totalTracks" variant="success")
            b-progress-bar(:value="state.trackIndex")
          .card-body
            .card-content
              h5.card-title {{ state.trackName }}
              div.info-table
                div.table-row(v-if="state.playlistName")
                  div.table-cell
                    i.fas.fa-list-ul
                  div.table-cell
                    p {{ state.playlistName }}
                div.table-row
                  div.table-cell
                    i.fas.fa-compact-disc
                  div.table-cell
                    p {{ state.albumName }}
                div.table-row
                  div.table-cell
                    i.fas.fa-user-friends
                  div.table-cell
                    p {{ state.artistName }}
                div.table-row
                  div.table-cell
                    i.fas.fa-hourglass-end
                  div.table-cell
                    p {{ state.progress | time }} / {{ state.duration | time }}
            .row.mt-2
              .col-lg-4.p-1
                b-button.btn-block(
                  @click="updatePlayerState(slotNumber)",
                  :disabled="!playbackDevice",
                  variant="primary"
                )
                  i.fas.fa-stop-circle
              .col-lg-4.p-1
                template(v-if="activeDevices.length > 1")
                  b-dropdown.btn-block(
                    split,
                    @click="restoreFromPlayerState(slotNumber)",
                    variant="success"
                  )
                    template(#button-content)
                      i.fas.fa-play-circle
                    b-dropdown-item.disabled Start playback on:
                    b-dropdown-divider
                    b-dropdown-item(
                      v-for="device in activeDevices",
                      @click="restoreFromPlayerState(slotNumber, device.id, device.name)",
                      :key="device.id"
                    ) {{ device.name }}
                template(v-else)
                  b-button.btn-block(
                    @click="restoreFromPlayerState(slotNumber)",
                    variant="success"
                  )
                    i.fas.fa-play-circle
              .col-lg-4.p-1
                b-button.btn-block(
                  @click="deletePlayerState(slotNumber)",
                  variant="danger"
                )
                  i.fas.fa-trash-alt
      a.floating-btn(@click="storePlayerState", :disabled="!playbackDevice")
        i.fas.fa-pause-circle
  div
</template>

<script>
export default {
  name: "Main",
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
      this.$api.fetchActiveDevices().then((activeDevices) => {
        this.activeDevices = activeDevices
        this.playbackDevice = undefined
        this.playbackDevicesInitiallyRequested = true

        activeDevices.forEach(device => {
          if (device.active) {
            this.playbackDevice = device
          }
        });
      }, (err) => {
        this.playbackDevice = undefined

        this.showErrorMessage("Failed to request active devices.")
        console.error("Failed to request actives devices from backend.", err)
      })
    },
    fetchPlayerStates: function () {
      this.$api.fetchPlayerStates().then((playerStates) => {
        this.playerStates = playerStates
      }, (err) => {
        this.showErrorMessage("Failed to request your player states.")
        console.error("Failed to request player states from backend.", err)
      })
    },
    updatePlayerState: function (slotNumber) {
      this.$api.updatePlayerState(slotNumber).then(() => {
        console.info(`Successfully updated player state in slot ${slotNumber}.`)
        this.fetchPlayerStates()
      }, (err) => {
        this.showErrorMessage("Failed to update player state.")
        console.error(`Failed to update player state in slot ${slotNumber}.`, err)
      })
    },
    storePlayerState: function () {
      this.$api.storePlayerState().then(() => {
        console.info("Successfully updated player state in new slot.")
        this.fetchPlayerStates()
      }, (err) => {
        this.showErrorMessage("Failed to store new player state.")
        console.error("Failed to store new player state.", err)
      })
    },
    deletePlayerState: function (slotNumber) {
      this.$api.deletePlayerState(slotNumber).then(() => {
        console.info(`Successfully deleted player state in slot ${slotNumber}.`)

        // Avoid re-fetching the player states, we can compute the change locally
        this.playerStates.splice(slotNumber, 1);
      }, (err) => {
        this.showErrorMessage("Failed to delete the player state.")
        console.error(`Failed to delete player state in slot ${slotNumber}.`, err)
      })
    },
    restoreFromPlayerState: function (slotNumber, deviceID, deviceName) {
      this.$api.restoreFromPlayerState(slotNumber, deviceID).then(() => {
        console.info(`Successfully restored player state from slot ${slotNumber} on device ${deviceID}.`)
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

      this.fetchActiveDevices()
      this.fetchPlayerStates()
    }, (err) => {
      this.showErrorMessage("Failed initializing the app. Please reload the page.")
      console.error("Failed fetching the CSRF token.", err)
    })
  }
}
</script>