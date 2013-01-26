angular.module('sallingshome', []).
        filter('relDate', function() {
        return function(dstr) {
            return moment(dstr).fromNow();
        };
    }).
    filter('calDate', function() {
        return function(dstr) {
            return moment(dstr).calendar();
        };
    }).
    filter('money', function() {
        return function(cents) {
            var x = parseInt(cents, 10) / 100;
            var valString = x.toFixed(2).toString().replace(/\B(?=(?:\d{3})+(?!\d))/g, ",");
            return "$" + valString;
        };
    }).
    config(['$routeProvider', '$locationProvider',
            function($routeProvider, $locationProvider) {
                $routeProvider.
                    when('/admin/', {templateUrl: '/static/partials/admin/index.html',
                                     controller: 'AdminCtrl'}).
                    when('/admin/topay/', {templateUrl: '/static/partials/admin/topay.html',
                                           controller: 'AdminToPayCtrl'}).
                    when('/admin/tasks/', {templateUrl: '/static/partials/admin/tasks.html',
                                           controller: 'AdminTasksCtrl'}).
                    when('/admin/users/', {templateUrl: '/static/partials/admin/users.html',
                                           controller: 'AdminUsersCtrl'}).
                    otherwise({redirectTo: '/admin/'});
                $locationProvider.html5Mode(true);
                $locationProvider.hashPrefix('!');
            }]);

function AdminCtrl($scope, $http) {
    $http.get("/api/currentuser/").success(function(data) {
        $scope.guser = data;
    });
}


function AdminToPayCtrl($scope, $http) {
    $http.get("/api/admin/topay/").success(function(data) {
        $scope.topay = data;
    });
}

function AdminTasksCtrl($scope, $http) {

    var calcTotal = function() {
        $scope.total = _.reduce($scope.tasks,
                                function(a, t) {
                                    return a + (t.value * 30 / t.period);
                                }, 0);
    };

    $scope.changedTask = _.debounce(function(t) {
        console.log("Changed", t);
        $scope.$apply(function() {
            $http.post("/api/admin/tasks/update/",
                       "taskKey=" + encodeURIComponent(t.Key) +
                       "&disabled=" + t.disabled +
                       "&name=" + encodeURIComponent(t.name) +
                       "&period=" + t.period + "&value=" + t.value,
                       {headers: {"Content-Type": "application/x-www-form-urlencoded"}}).
                success(function(e) {
                    t.editing = false;
                });
            calcTotal();
        });
    }, 3000);

    $scope.$watch('tasks', calcTotal);

    $http.get("/api/admin/tasks/").success(function(data) {
        $scope.tasks = data;

    });
    $http.get("/api/admin/users/").success(function(data) {
        $scope.users = data;
    });

}

function AdminUsersCtrl($scope, $http) {
    $http.get("/api/admin/users/").success(function(data) {
        $scope.users = data;
    });
}
