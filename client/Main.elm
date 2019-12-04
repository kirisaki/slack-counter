module Main exposing (main)

import Browser exposing (Document, UrlRequest)
import Browser.Navigation as Nav exposing (Key)
import Html exposing (..)
import Html.Attributes exposing (..)
import Html.Events exposing (..)
import Maybe
import Url exposing (Url)
import Url.Parser as UP exposing ((</>), Parser)


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


type alias Model =
    { key : Key
    , route: Route
    }


type Route
    = Index
    | NotFound


type Language
    = Japanese
    | English


type Msg
    = NoOp


init : () -> Url -> Key -> ( Model, Cmd Msg )
init _ _ k =
    ( { key = k
      , route = Index
      }
    , Cmd.none
    )

update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case ( msg, model.route ) of
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
        [ div [ classList [ ("container", True)
                          ]
              ]
              [ h1 [] [ text "nyaan" ]
              ]
        ]
    }

notFound : Model -> Html Msg
notFound model =
    text "nyaan..."
