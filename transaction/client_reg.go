package transaction

import (
    "time"
    "fmt"
    "github.com/discoviking/fsm"
    "github.com/stefankopieczek/gossip/log"
)

// SIP Client Transaction FSM
// Implements the behaviour described in RFC 3261 section 10

// FSM States
const (
    reg_client_state_init = iota
    reg_client_state_401
    reg_client_state_success
    reg_client_state_terminated
)

// FSM Inputs
const (
    reg_client_input_200 fsm.Input = iota
    reg_client_input_401
    reg_client_input_else
    reg_client_input_timer_a
    reg_client_input_timer_b
    reg_client_input_timer_d
    reg_client_input_transport_err
    reg_client_input_delete
)

func (tx *ClientTransaction) initRegisterFSM() {
        
    // Resend the request.
    init_resend := func() fsm.Input {
        fmt.Println("[client_reg]:Resending the request")
        tx.timer_a_time *= 2
        tx.timer_a.Reset(tx.timer_a_time)
        tx.resend()
        return fsm.NO_INPUT
    }

    // Just pass up the latest response.
    // act_passup := func() fsm.Input {
    //     tx.passUp()
    //     return fsm.NO_INPUT
    // }

    // Handle 300+ responses.
    // Pass up response and send ACK, start timer D.
    act_300 := func() fsm.Input {
        fmt.Println("[client_reg]:act_300")
        tx.passUp()
        tx.Ack()
        if tx.timer_d != nil {
            tx.timer_d.Stop()
        }
        tx.timer_d = time.AfterFunc(tx.timer_d_time, func() {
            tx.fsm.Spin(reg_client_input_timer_d)
        })
        return fsm.NO_INPUT
    }

    // Send up transport failure error.
    act_trans_err := func() fsm.Input {
        fmt.Println("[client_reg]: act_trans_err")
        tx.transportError()
        return reg_client_input_delete
    }

    // Send up timeout error.
    act_timeout := func() fsm.Input {
        fmt.Println("[client_reg]: act_timeout")
        tx.timeoutError()
        return reg_client_input_delete
    }

    // Pass up the response and delete the transaction.
    act_passup_delete := func() fsm.Input {
        fmt.Println("[client_reg]: act_passup_delete")
        tx.passUp()
        tx.Delete()
        return fsm.NO_INPUT
    }

    // Just delete the transaction.
    act_delete := func() fsm.Input {
        fmt.Println("[client_reg]: act_delete")
        tx.Delete()
        return fsm.NO_INPUT
    }

    // TODO: Sushant
    // write act_auth
    act_auth := func() fsm.Input{
        fmt.Println("[client_reg]: act_auth")
        tx.Auth()
        return fsm.NO_INPUT
    }

    // Define States

    // Initial State
    reg_client_state_def_init := fsm.State{
        Index: reg_client_state_init,
        Outcomes: map[fsm.Input]fsm.Outcome{
            reg_client_input_401:           {reg_client_state_401, act_auth},
            reg_client_input_200:           {reg_client_state_terminated, act_passup_delete},
            reg_client_input_else:          {reg_client_state_terminated, act_300},
            reg_client_input_timer_a:       {reg_client_state_init, init_resend},
            reg_client_input_timer_b:       {reg_client_state_terminated, act_timeout},
            reg_client_input_transport_err: {reg_client_state_terminated, act_trans_err},
        },
    }

    // 401 Unauth State
    reg_client_state_def_401 := fsm.State{
        Index: reg_client_state_401,
        Outcomes: map[fsm.Input]fsm.Outcome{
            reg_client_input_401:      {reg_client_state_401, act_auth},
            reg_client_input_200:           {reg_client_state_success, fsm.NO_ACTION},
            reg_client_input_else:          {reg_client_state_terminated, act_300},
            reg_client_input_timer_a:       {reg_client_state_401, act_auth},
            reg_client_input_timer_b:       {reg_client_state_terminated, act_timeout},
            reg_client_input_transport_err: {reg_client_state_terminated, act_trans_err},
        },
    }

    // Completed
    reg_client_state_def_success := fsm.State{
        Index: reg_client_state_success,
        Outcomes: map[fsm.Input]fsm.Outcome{
            reg_client_input_401:           {reg_client_state_success, fsm.NO_ACTION},
            reg_client_input_200:           {reg_client_state_success, fsm.NO_ACTION},
            reg_client_input_else:          {reg_client_state_success, fsm.NO_ACTION},
            reg_client_input_timer_d:       {reg_client_state_terminated, act_delete},
            reg_client_input_transport_err: {reg_client_state_terminated, act_trans_err},
            reg_client_input_timer_a:       {reg_client_state_success, fsm.NO_ACTION},
            reg_client_input_timer_b:       {reg_client_state_success, fsm.NO_ACTION},
        },
    }

    // Terminated
    reg_client_state_def_terminated := fsm.State{
        Index: reg_client_state_terminated,
        Outcomes: map[fsm.Input]fsm.Outcome{
            reg_client_input_401:      {reg_client_state_terminated, fsm.NO_ACTION},
            reg_client_input_200:      {reg_client_state_terminated, fsm.NO_ACTION},
            reg_client_input_else:     {reg_client_state_terminated, fsm.NO_ACTION},
            reg_client_input_timer_a:  {reg_client_state_terminated, fsm.NO_ACTION},
            reg_client_input_timer_b:  {reg_client_state_terminated, fsm.NO_ACTION},
            reg_client_input_delete:   {reg_client_state_terminated, act_delete},
        },
    }

    fsm, err := fsm.Define(
        reg_client_state_def_init,
        reg_client_state_def_401,
        reg_client_state_def_success,
        reg_client_state_def_terminated,
    )

    if err != nil {
        log.Severe("Failure to define REGISTER client transaction fsm: %s", err.Error())
    }
    fmt.Println("REGISTER FSM created")

    tx.fsm = fsm
}

