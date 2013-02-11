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
    $scope.total = 0;

    $scope.updatePaying = function(ob) {
        $scope.total = _.reduce($scope.topay,
                                function(a, t) {
                                    return a + (t.paying ? t.amount : 0);
                                }, 0);
    };

    $http.get("/api/admin/topay/").success(function(data) {
        $scope.topay = data;
        _.each($scope.topay, function(e) {e.paying = false;});
    });
}

function AdminTasksCtrl($scope, $http) {

    var calcTotal = function() {
        $scope.total = _.reduce($scope.tasks,
                                function(a, t) {
                                    return a + (t.value * 30 / t.period);
                                }, 0);
    };

    $scope.changedTask = function(t) {
        console.log("Changed", t);
        $http.post("/api/admin/tasks/update/",
                   "taskKey=" + encodeURIComponent(t.Key) +
                   "&disabled=" + t.disabled +
                   "&name=" + encodeURIComponent(t.name) +
                   "&description=" + encodeURIComponent(t.description) +
                   "&period=" + t.period + "&value=" + t.value,
                   {headers: {"Content-Type": "application/x-www-form-urlencoded"}}).
            success(function(e) {
                t.editing = false;
            });
        calcTotal();
    };

    $scope.makeAvailable = function(t) {
        $http.post("/api/admin/tasks/makeAvailable/",
            "taskKey=" + encodeURIComponent(t.Key),
            {headers: {"Content-Type": "application/x-www-form-urlencoded"}}).
            success(function(e) {
                t.next = e.next;
            });
    };

    $scope.makeUnavailable = function(t) {
        $http.post("/api/admin/tasks/makeUnavailable/",
            "taskKey=" + encodeURIComponent(t.Key),
            {headers: {"Content-Type": "application/x-www-form-urlencoded"}}).
            success(function(e) {
                t.next = e.next;
            });
    };

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
