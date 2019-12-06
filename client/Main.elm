module Main exposing (main)

import Browser exposing (Document, UrlRequest)
import Browser.Navigation as Nav exposing (Key)
import Html exposing (..)
import Html.Attributes exposing (..)
import Html.Events exposing (..)
import Maybe
import Url exposing (Url)
import Url.Parser as UP exposing ((</>), Parser)
import Http
import Json.Decode as JD
import Task
import List
import Time
import String

main : Program () Model Msg
main =
    Browser.application
        { init = init
        , view = view
        , update = update
        , subscriptions = subscriptions
        , onUrlRequest = (\l -> NoOp)
        , onUrlChange = (\u -> NoOp)
        }

type alias Activity =
    { start : Int
    , activity : List Int
    }

type alias Activities = List Activity

type alias Model =
    { key : Key
    , route : Route
    , activities : Activities
    }


type Route
    = Index
    | NotFound


type Language
    = Japanese
    | English


type Msg
    = GetActivities (Result Http.Error Activities)
    | NoOp

getActivities : Cmd Msg
getActivities =
    let
        dec = JD.field "activities" (JD.list acdec)
        acdec = JD.map2 Activity
                (JD.field "start" JD.int)
                (JD.field "activity" (JD.list JD.int))
    in
        Http.get
            { url = "/query?duration=14"
            , expect = Http.expectJson GetActivities dec
            }

init : () -> Url -> Key -> ( Model, Cmd Msg )
init _ _ k =
    ( { key = k
      , route = Index
      , activities = []
      }
    , getActivities
    )

update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        GetActivities res ->
            case res of
                Ok a ->
                    ({ model | activities = a } , Cmd.none)
                Err e ->
                    let
                        _ = Debug.log "http" e
                    in
                        (model, Cmd.none)
        _ ->
            ( model
            , Cmd.none
            )


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.none


view : Model -> Document Msg
view model =
    { title = "slack-counter"
    , body =
        [ div [ class "container" ]
              [ h1 [] [ text "slack-counter" ]
              , table [ class "activity" ] [ thead [] ([td [] []] ++ (List.map (\m -> td [] [ text (String.fromInt m) ]) (List.range 0 23)))
                                           , tbody [] (rows model)]
              ]
        ]
    }

rows : Model -> List (Html Msg)
rows model = List.map (\row -> tr [ class "column" ] ([ date row.start ] ++ cells row.activity))  model.activities

date : Int -> Html Msg
date mills =
    let
        posix = Time.millisToPosix mills
        year = Time.toYear Time.utc posix
        month = case Time.toMonth Time.utc posix of
                    Time.Jan -> "1"
                    Time.Feb -> "2"
                    Time.Mar -> "3"
                    Time.Apr -> "4"
                    Time.May -> "5"
                    Time.Jun -> "6"
                    Time.Jul -> "7"
                    Time.Aug -> "8"
                    Time.Sep -> "9"
                    Time.Oct -> "10"
                    Time.Nov -> "11"
                    Time.Dec -> "12"
        day = Time.toDay Time.utc posix
           
    in
        text (String.fromInt year ++ "-" ++ month ++ "-" ++ String.fromInt day)

cells : List Int -> List (Html Msg)
cells xs = List.map (\cell -> td [ class "cell" ] [ text (String.fromInt cell )]) xs

notFound : Model -> Html Msg
notFound model =
    text "nyaan..."
